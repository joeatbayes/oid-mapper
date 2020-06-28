package main
 
import (
	"bufio"
	"fmt"
	"log"
	"os"
	//"strings"
	"sort"
	//"strconv"
	"path/filepath"
	"sync"
	"runtime"
	"time"
)

/* Merge pre-sorted input files into a sorted output file.   This technique is based roughly
 on the bisect file pattern used in my DEM waterflow array ported from python to go.
 Some ideas borrowed from bibliographic indexing along with rway merge. 
 Seeing if go can come close to the native linux sort. 
 
 TODO:  Reduce the number of sorts to reduce total time and drive
   the hard disks at closer to capacity.  In early tests the 
   drive capacity is closer to 450MiB/s and we are at best driving
   about 90MiB/s with with the last pass at 27MiB/s. 
   Read a block of lines from each file
   sort all the blocks together.   Only when processing a block with a 
   string greater than the last string read from a given file do we need
   to readd and sort again. With random distribution ad data between files 
   this should allow us to process nearly the entire block without the 
   next sort. It could require scanning the entire file list to find the
   next violator but we could do this with a single sort and just remember
   the lowest string from any of the files since we know it will be the 
   first to be violated.  When we violate the rule of processing a string
   that is greater than the smallest next string from any file we need to 
   read another block from that file and resort.     It should dramatically
   reduce our sorts which should allow us to drive the drives at closer
   to their IO capacity.
*/

type SortLine struct {
	lastString string
	fp *os.File 
	scanner *bufio.Scanner
	eof bool
	readCnt int
}

type SortLines []*SortLine
const MaxOpenFiles = 300
var MaxMergeThreadsActive = 18

var procSortLinesWG sync.WaitGroup
var openFileCnt = 0
var countLock sync.Mutex
var mergeThreadsActive = 0
func fileExists(name string) (bool) {
  _, err := os.Stat(name)
  if os.IsNotExist(err) {
    return false
  }
  return err != nil
}

// open all the files names encapsulate them
// in a sort spec and return a slice with all 
// the specs.
func makeSortLines(fnamesIn []string) SortLines {
	specs := make([]*SortLine, 0)
	for ndx:=0; ndx < len(fnamesIn); ndx++ {
		fiName := fnamesIn[ndx]
		//fmt.Println("L37: makeSortLines fiName=", fiName)
		fiPtr, err := os.Open(fiName)
		if err != nil {
			log.Fatalf("failed opening file %s: %s", fiName, err)
		} else {
			scanner := bufio.NewScanner(fiPtr)
			aspec := &SortLine {"", fiPtr, scanner, false,0}
			specs = append(specs, aspec);
		}
	}
	return specs
}

func (sl SortLines)sortSortLines() {
	if len(sl) > 1 {
		sort.Slice(sl, func(i, j int) bool { return sl[i].lastString < sl[j].lastString })
	}
}

func (sel *SortLine) readNext() string {
	if sel.eof == true {
		sel.lastString = "~~"
		fmt.Println("L57 EOF", " name=", sel.fp.Name())
	} else {
		sel.readCnt += 1
		more := sel.scanner.Scan()
		if more == false {
			sel.eof = true
		} 
		sel.lastString = sel.scanner.Text()
		//fmt.Println("L63: name=", sel.fp.Name(), " sel.lastString=", sel.lastString)
	}
	return sel.lastString
}


func (sln SortLines) dumpAll() { 
	for ndx:=0; ndx < len(sln); ndx ++ {
		sel := sln[ndx]
		fmt.Println("L74 dumpAll: ndx=", ndx,  " cnt=", sel.readCnt, " fi=", sel.fp.Name(), " eof=", sel.eof, " txt=", sel.lastString)
	}
}

/* Read the next line for all files in the sort line spec */
func (sln SortLines) readNextAll() SortLines {
	for ndx:=0; ndx < len(sln); ndx ++ {
		sel := sln[ndx];
		sel.readNext();
	}
	//fmt.Println("L85:End readNextAll DUMP")
	//sln.dumpAll()
	sln.sortSortLines();
	//  Handle special case of some files
	//  hitting EOF and needing to be removed
	//  from the set.
	for len(sln) > 0 {
		lastndx := len(sln) -1
		if sln[lastndx].lastString == "~~" {
			sln[lastndx].fp.Close()
			sln = sln[ : lastndx] // pop last element
		} else {
			break;
		}
	}
	//fmt.Println("L100:End readNextAll DUMP")
	//sln.dumpAll()
	return sln
}

/* Read strings from a set of files concurrently open
find the lowest avaialble string from all files and then
read the next lowest string.    Read a string from each 
file.  Sort them into ascending order.  Take the first
record and write to the file.  Read a line from the file
chosen and as long as it is less than value in second 
entry can just record.  If it is greater than value
in second entry must resort and go again.  This is intended
to test the idea that sorting a small array that doesn't change
much possibly in the worst case of num of lines in every file 
is faster than read-readng files more times to allow simple
two line compare. */
func mergeFiles(fnameOut string, fnamesIn []string) {
	// TODO: Handle special case with only 1 input file
	sln := makeSortLines(fnamesIn);
	sln = sln.readNextAll()
	sln.sortSortLines()
	selected := sln[0]
	next := selected
	if len(fnamesIn) > 1 {
	  next = sln[1]
	}
	lastWrite := ""
	lineCnt := 0
	sinceStatCnt := 0
	
	fout, foerr := os.OpenFile(fnameOut, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
			log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	for {
		lineCnt += 1
		sinceStatCnt += 1

		if len(sln) == 0 {
			// all files have been removed from consideration
			break
		}
		
		if (len(sln) == 1) {
			// Handle Special case of only 1 input file remaining
			for  selected.lastString != "~~" {
				fout.WriteString(selected.lastString)
				fout.WriteString("\n")
				lastWrite = selected.lastString
				selected.readNext()
			}
			break;
		}
		
		//sln.dumpAll() 
		if selected.lastString > next.lastString {
			sln.sortSortLines()
			selected = sln[0]
			next = sln[1]
		}
		//if sinceStatCnt > 5000000 {
		//	sinceStatCnt = 0
		//	fmt.Println("L164: statCnt lineCnt=", lineCnt, "sel=", selected.lastString, " next=", next.lastString)
		//}
		if lastWrite != selected.lastString {
			// handle special case of duplicate line that should be filtered out.
			fout.WriteString(selected.lastString)
			fout.WriteString("\n")
			lastWrite = selected.lastString
		}
		// Read the next line from the file we read the last one 
		// from. 
		selected.readNext()
		if (selected.lastString == "~~") {
			// Last string from our selected file indicates
			// EOF so we remove it from the list of files
			// to consider. 
			selected.fp.Close()
			if len(sln) < 1 {
				break
			}
			sln = sln[1:]
			sln.sortSortLines()
			selected = sln[0]
			if len(sln) < 2 {
				next = nil
			} else {
			  next = sln[1]
			}
		}
	}
	// TODO:remove the segment files. 
	datawriter.Flush()
	countLock.Lock()
		openFileCnt -= len(fnamesIn)
		mergeThreadsActive -= 1
	countLock.Unlock()
	procSortLinesWG.Done()
}


func makeSegFiName(baseName string,  segLev int, suff string) string{
	return baseName + "-" + fmt.Sprintf("s-%03d",segLev) + "." + suff
}

func makeBatchSegFiName(baseName string, segLev int, batchCnt int,  suff string) string {
	return makeSegFiName(fmt.Sprintf("%sb-%04d.merge", baseName, batchCnt), segLev, suff)
}

/* Merge a group of files and return the list of 
  of files produced.  Breaks the input set up based
  on the maximum number of files specified. 
  */
func mergFilesList(flist []string, baseName string, segLev int, suff string,  maxConcurrentSegPerThread int) [] string {
	foutCnt := 0
	if len(flist) <= 1 {
		return flist
	}
	filesWritten := make([]string,0)
	var fnames = make([]string,0)
	batchCnt := 0
	flush := func () {
		batchCnt += 1
		foutCnt+= 1
		foutName := makeBatchSegFiName(baseName, segLev,  batchCnt, suff)
		fmt.Println("L173: MergeSet=", fnames, " foutName=", foutName)
		filesWritten = append(filesWritten, foutName)
		procSortLinesWG.Add(1)
		waitCnt := 0
		for (mergeThreadsActive + 1) > MaxMergeThreadsActive {
			waitCnt += 1
			fmt.Println("L250: Too many active merg threads waitCnt=", waitCnt, " MaxMergeThreadsActive=", MaxMergeThreadsActive, " mergeThreadsActive=", mergeThreadsActive, "sleeping")
			time.Sleep(time.Second)
		}
		for (openFileCnt + len(fnames)) > MaxOpenFiles {
			waitCnt += 1
			fmt.Println("L255: Too many files open so wait MaxOpenFiles=", MaxOpenFiles, " openFileCnt=", openFileCnt)
			time.Sleep(time.Second)
		}
		fmt.Println("L262: OK to process MaxOpenFiles=", MaxOpenFiles, " openFileCnt=", openFileCnt, 
		countLock.Lock()
			openFileCnt += len(fnames)
			mergeThreadsActive += 1
		countLock.Unlock()
		
		// Start the GO Routine / Thread to do the actual processing
		go mergeFiles(foutName, fnames)
		fnames = make([]string,0)
	}
	// Build and execute batches
	for ndx := 0; ndx < len(flist); ndx++ {
		fInName := flist[ndx]
		fmt.Println("L174 file =", fInName, "batchCnt=", batchCnt)
		fnames = append(fnames, fInName)
		if len(fnames) >= maxConcurrentSegPerThread {
			flush()
		}
	}
	// Flush any for our last set.
	if len(fnames) > 0 {
		flush()
	}
	fmt.Println("L284 Waiting for Merge to finish batchCnt=", batchCnt, " segLev=", segLev, " baseName=", baseName, " MaxMergeThreadsActive=", MaxMergeThreadsActive, " mergeThreadsActive=", mergeThreadsActive," MaxOpenFiles=", MaxOpenFiles, " openFileCnt=", openFileCnt)
	procSortLinesWG.Wait()
	return filesWritten
}


func main() {
	maxConcurrentSegPerThread := 25
	concurrentSegPerThread := maxConcurrentSegPerThread
	portCoreToUse := 1.3
	numCore := int(float64(runtime.NumCPU()) * portCoreToUse)
	if numCore < 2 { numCore = 2}
	MaxMergeThreadsActive = numCore + 1 // reset starting default for current machine config
	suffix := "seg"
	fmt.Println("os.Args=", os.Args)
	if len(os.Args) < 3 {
		fmt.Println("Arg 1 must be input file name and arg2 must be output base name")
		panic("abort")
	}
	
	baseName := os.Args[1]
	segLevel := 0
	fOutName := os.Args[2]
	fmt.Println("baseName=", baseName)
	
	if fileExists(fOutName) {
		err := os.Remove(fOutName)
		if err != nil {
			fmt.Println("ERROR could not remove old output file ", fOutName, " err=", err)
		}
	}
	
	
	globPat := baseName + "*-" + fmt.Sprintf("%03d",0) + "." + suffix
	fmt.Println("L161: globPat=", globPat)
	// Get a list of files
	fileList, _ := filepath.Glob(globPat)
	if fileList == nil { 
		fmt.Println("L322 Found no files for globPat=", globPat)
	}
	fmt.Println("L132: initial fileList len=", len(fileList)," files=", fileList)
	// re-process merge files until we run out we consoldate them to a single 
	// file 
	for len(fileList) > 2 {
		segLevel+=1
		fmt.Println("L292: MergeSet Phase ", segLevel, " numFi=", len(fileList), "files=", fileList)
		// Adjust MaxFilesPerThread and MaxThreads based on
		// number of cores.   In general 
		if len(fileList) <= maxConcurrentSegPerThread {
			// If we have few enough segments to wrap up in this pass
			// then do so. 
			concurrentSegPerThread = maxConcurrentSegPerThread
		} else {
			// Too many to wrap up in a single pass so 
			// group he files evently between the available cores
			concurrentSegPerThread = (len(fileList) / numCore)+1
			if  concurrentSegPerThread > maxConcurrentSegPerThread {
				concurrentSegPerThread = maxConcurrentSegPerThread
			}
		}
		fileList = mergFilesList(fileList, baseName, segLevel, "seg" ,  concurrentSegPerThread)
		fmt.Println("L294: Waiting on Threads for files=", fileList)
		fmt.Println("L331: finished waiting on threads start next phase")
	}
	fmt.Println("L336: Renaming ", fileList[0], " to ", fOutName)
	err := os.Rename(fileList[0], fOutName)
	if err != nil {
	   fmt.Println(" Error renaming file segment from ", fileList[0], " to ", fOutName, " err=", err)
	}
}