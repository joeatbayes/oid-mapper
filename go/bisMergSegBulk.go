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
 
 Adapted from Nieve approach to read larger bulks of data and reduce 
 total run time by reducing sorts. 
 
 Reduce the number of sorts to reduce total time and drive
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
 to their IO capacity.   Always read the next block for lowest file
 in a separate thread and pre-sort it before merging into the complete set.
 
*/

type SortLine struct {
	text string
	fp *os.File 
	scanner *bufio.Scanner
	eof bool
	readCnt int
}

type SortLines struct {
	lines []*SortLine
	maxBytesPerRead int
}


const MaxOpenFiles = 300
var MaxMergeThreadsActive = 18
var GMaxBytesPerRead = 32000   // Warning smallest total buffer size will be 
                               // number of active threads * this number
                               // In some instances we will have to read
                               // forward in the files so buffer could grow
                               // beyond this minimum. 

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
func makeSortLines(fnamesIn []string) *SortLines {
	specs := make([]*SortLine, 0)
	sln := SortLines {specs, GMaxBytesPerRead}
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
	sln.lines = specs
	return &sln
}

func (sl *SortLines) sortSortLines() {
	lines := sl.lines
	if len(lines) > 1 {
		sort.Slice(lines, func(i, j int) bool { return lines[i].text < lines[j].text })
	}
}

func (sel *SortLine) readNextChunck(maxBytes int) []string {
	tout := make([]string,0,500)
	bytesRead := 0
	line := "~~"
	if sel.eof == true {
			sel.text = "~~"
			fmt.Println("L57 EOF", " name=", sel.fp.Name())
	} else {
		for bytesRead < maxBytes  {
			sel.readCnt += 1
			more := sel.scanner.Scan()
			line = sel.scanner.Text()
			bytesRead += len(line)
			tout = append(tout, line)
			if more == false {
				sel.eof = true
				break
			} 
			//fmt.Println("L63: name=", sel.fp.Name(), " sel.text=", sel.text)
		}
		sel.text = line
	}
	return tout
}

func (sln *SortLines) dumpAll() { 
	for ndx:=0; ndx < len(sln.lines); ndx ++ {
		sel := sln.lines[ndx]
		fmt.Println("L74 dumpAll: ndx=", ndx,  " cnt=", sel.readCnt, " fi=", sel.fp.Name(), " eof=", sel.eof, " txt=", sel.text)
	}
}

/* Return the SortLine with the lowest last read 
  line of text by sort order. If no sort line has
  sort order less than "~~" then will return the 
  the first sort line even if it is eof.  This can
  be used as indication the process is complete */
func (sln *SortLines) findLowest() (*SortLine, *SortLine, bool) {
	slines := sln.lines
	lowest := slines[0]
	highest:= slines[0]
	allEOF := true
	for ndx:=0; ndx < len(slines); ndx ++ {
		sel := slines[ndx];
		if sel.eof == false {
			allEOF = false
			if sel.text < lowest.text {
				lowest = sel
			}
			if sel.text > highest.text {
				highest = sel
			}
		}
	}
	return lowest, highest, allEOF
}

/* Read the next chunk of lines from all files
  and adds then to the sln buffer.  Also sets 
  the the sln.lowest based on the last line read
  from all files that has the lowest sort order */
func (sln *SortLines) readNextAll() []string {
	slines := sln.lines
	buf := make([]string,0,5000)
	//fmt.Println("L172 len(slines)=", len(slines), " slines=", slines)
	for ndx :=0; ndx < len(slines); ndx ++ {
		sel := slines[ndx];
		//fmt.Println("L174: sel=", sel)
		if sel.eof == false {
			lines := sel.readNextChunck(sln.maxBytesPerRead);
			//fmt.Println("L177: lastText = ", sel.text,  " numLines=", len(lines))
			buf = append(buf, lines...)
		}
	}
	// Don't sort here because we will need to sort
	// the read back into the larger buffer anyway
	return buf
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
	sln := makeSortLines(fnamesIn);
	fmt.Println("L204: fnameOut=", fnameOut, " fnamesIn=", fnamesIn);
	buff := sln.readNextAll() // preload buffer 
	if len(buff) > 0 {
	   sort.Strings(buff)
	} else {
		fmt.Println("L185: ReadNextAll: no lines available")
	}
	lowest, _, allEOF := sln.findLowest()
	lowestStr := lowest.text
	
	//fmt.Println("L206: after readNextAll sln=", sln)
	lastWrite := ""
	lineCnt := 0
	fout, foerr := os.OpenFile(fnameOut, os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
			log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	rowNdx := 0
	sinceLastRead := 0
	for rowNdx = 0; rowNdx < len(buff); rowNdx++ {
		sinceLastRead += 1
		lineCnt += 1
		lstr := buff[rowNdx]
		if lstr <= lowestStr || allEOF == true{
			if lstr != lastWrite {
				fout.WriteString(lstr)
				fout.WriteString("\n")
				lastWrite = lstr
			}
		} else {
			//fmt.Println("L229: ReadNext rowNdx=", rowNdx, " bufLen=", len(buff), " sinceLastRead=", sinceLastRead, " lstr=", lstr[:12], " lowestStr=", lowestStr[:12])
			// The line we are reading is greater than 
			// the lowest sort order of the last lines 
			// read from files so we need to read another
			// chunk from that file.
			buff = buff[rowNdx:] // truncate the buffer to remove strings already written
			rowNdx = 0 // reset counter to accomodate truncated buffer
			
			// NOTE:  Would it be faster to load each segment
			//  since we know they are in order we should be able 
			//  to process from 1 buffer until we know there is 
			//  a buffer that has a higher value then transition
			//  up it. We would have to traverse the buffers to 
			//  find the one that has the lowest order string 
			//  capable that fits but if we have a number of strings
			//  smaller than the next buffer we could avoid the 
			//  scan.  Problem is that we would need to scan numFileSeg
			//  for the recovery but that may be cheaper than attempting
			//  to sort a much larger list of records loaded with this 
			//  approach we could use background threads to read and load
			//  next segments for each file buffer. 
			//  
			// TODO: PUSH READ ALL NEXT OUT TO SEPARATE THREAD AS LONG
			// as length of buffer loaded is less than a MaxBufLen
			// then should allow a second thread to be loading 
			// and presorting the next segment. 
			var lines []string
			if len(buff) < 200000 {
				// TODO: Find a way to allow other threads 
				//  to read the input files, combine them 
				//  and pre-sort them so we have them in 
				//  easy to consume fashion when we need
				//  the next one. Could keep and array
				//  of several pre-loaded so we can just
				//  pop it off the stack when we want the next one. 
				 
				// If buffer is less than a configurable threashold
				// then allow readNextAll so we can sort a larger chunk once.
				lines = sln.readNextAll()
			} else {
				lines = lowest.readNextChunck(sln.maxBytesPerRead);
			}
			lowest, _, allEOF = sln.findLowest()
			if len(lines) > 0 {
					buff = append(buff, lines...)
					sort.Strings(buff)
			} 
			lowestStr = lowest.text
			sinceLastRead = 0
		}
	} // for
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
		fmt.Println("L262: OK to process MaxOpenFiles=", MaxOpenFiles, " openFileCnt=", openFileCnt)
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
	maxConcurrentSegPerThread := 400
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