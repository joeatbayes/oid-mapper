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
)

/* Merge pre-sorted input files into a sorted 
 output file.   This technique is based roughly on the bisect file
 pattern used in my DEM waterflow array ported from python to go.
 Some ideas borrowed from bibliographic indexing along with rway merge. 
 Seeing if go can come close to the native linux sort. */

type SortLine struct {
	lastString string
	fp *os.File 
	scanner *bufio.Scanner
	eof bool
	readCnt int
}

type SortLines []*SortLine

var procSortLinesWG sync.WaitGroup

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
   sort.Slice(sl, func(i, j int) bool { return sl[i].lastString < sl[j].lastString })
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
	next := sln[1]
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
		if sinceStatCnt > 5000000 {
			sinceStatCnt = 0
			fmt.Println("L164: statCnt lineCnt=", lineCnt, "sel=", selected.lastString, " next=", next.lastString)
		}
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
	datawriter.Flush()
	procSortLinesWG.Done()
}


func makeSegFiName(baseName string,  segLev int, suff string) string{
	return baseName + "-" + fmt.Sprintf("%03d",segLev) + "." + suff
}

/* Merge a group of files and return the list of 
  of files produced.  Breaks the input set up based
  on the maximum number of files specified. 
  
  TODO:  To control total file handle usage we need
  to detect how many we have spawned and limit 
  any more until the last ones have been completed.
   */
func mergFilesList(flist []string, baseName string, segLev int, suff string,  maxConcurrentSegPerThread int) [] string {
	foutCnt := 0
	if len(flist) <= 1 {
		return flist
	}
	filesWritten := make([]string,0)
	var fnames = make([]string,0)
	batchCnt := 0
	for ndx := 0; ndx < len(flist); ndx++ {
		fInName := flist[ndx]
		fmt.Println("L174 file from Glob=", fInName)
		fnames = append(fnames, fInName)
		if len(fnames) >= maxConcurrentSegPerThread {
			batchCnt += 1
			fmt.Println("L170: MergeSet=", fnames)
			foutCnt+= 1
			foutName := makeSegFiName(baseName + fmt.Sprintf("%04d.merge", batchCnt), segLev + 1, suff)
			fmt.Println("L173: foutName=", foutName)
			filesWritten = append(filesWritten, foutName)
			procSortLinesWG.Add(1)
			go mergeFiles(foutName, fnames)
			fnames = make([]string,0)
		}
	}
	
	if len(fnames) > 0 {
		batchCnt += 1
		foutName := makeSegFiName(baseName + fmt.Sprintf("%04d.merge", batchCnt), segLev + 1, suff)
		procSortLinesWG.Add(1)
		go mergeFiles(foutName, fnames)
		filesWritten = append(filesWritten, foutName)
	}
	
	// TODO:Merge the files into a larger file 
	// TODO:remove the segment files. 
	return filesWritten
}

/* Based on a glob pattern find the set of smallest 
 available files that meet the glob pattern and then
 merge them together opening up to maxConcurrentSegPerThread
 together. */
func mergFilesPat(baseName string, segLev int, suff string,  maxConcurrentSegPerThread int) [] string{
	globPat := baseName + "*-" + fmt.Sprintf("%03d",segLev) + "." + suff
	fmt.Println("L161: globPat=", globPat)
	// Get a list of files
	filesWritten := make([]string,0)
	flist, _ := filepath.Glob(globPat)
	if flist == nil { return filesWritten}
	return mergFilesList(flist, baseName, segLev, suff ,  maxConcurrentSegPerThread)
}


/*  Name the larger file with a new name eg if
 first segments input are ".seg" then name 
 the output to ".segp1" in pass 1,  then
 segp2 in pass 2,  etc.  Keep increasing
 the segment phase until we end up in a segment
 with only a single output file which is the
 one we want to keep file */
func mergSegFiles(globPatt string, startPhase int, maxConcurrentSeg int) {
	
}

func fileExists(name string) (bool) {
  _, err := os.Stat(name)
  if os.IsNotExist(err) {
    return false
  }
  return err != nil
}

func main() {
	maxConcurrentSegPerThread := 30
	portCoreToUse := 0.8
	numCore := int(float64(runtime.NumCPU()) * portCoreToUse)
	if numCore < 2 { numCore = 2}
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
	
	// TODDO: finish section to limit number of concurrent
	//  processors to 80% of available cores. 
	
	// TODO: Adjust MaxFilesPerThread and MaxThreads based on
	// number of cores.   In general we want to group
	// the files evently between the available cores
	concurrentSegPerThread := (len(filesWritten) / numCore)+1
	if  concurrentSegPerThread > maxConcurrentSegPerThread {
	  concurrentSegPerThread = maxConcurrentSegPerThread
	}
	filesWritten := mergFilesPat(baseName , 0, "seg",  concurrentSegPerThread)
	fmt.Println("L263: Waiting on Threads")
	procSortLinesWG.Wait()
	fmt.Println("L132: initial files numWritten=", len(filesWritten)," written=", filesWritten)
	// re-process merge files
	// until we run out we consoldate them to a single 
	// file 
	for len(filesWritten) > 2 {
		segLevel+=1
		fmt.Println("L292: Phase ", segLevel, " numFi=", len(filesWritten), "files=", filesWritten)
		concurrentSegPerThread = (len(filesWritten) / numCore)+1
		if  concurrentSegPerThread > maxConcurrentSegPerThread {
			concurrentSegPerThread = maxConcurrentSegPerThread
		}
		filesWritten = mergFilesList(filesWritten, baseName, segLevel, "seg" ,  concurrentSegPerThread)
		fmt.Println("L294: Waiting on Threads")
		procSortLinesWG.Wait()
	}
	
	err := os.Rename(filesWritten[0], fOutName)
	if err != nil {
	   fmt.Println(" Error renaming file segment from ", filesWritten[0], " to ", fOutName, " err=", err)
	}
}