package main

import (
	"bufio"
	"fmt"
	"log"
	"os"

	//"strings"
	//"sort"
	//"strconv"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

/* Merge pre-sorted input files into a sorted output file.  Attempt to
see if using GoLang we can devise a method that equals or beats
the performance of the linux sort. For 100GB size range files. 

The sorted files are needed to use technique is based roughly
on the bisect file pattern which is essentially a binary search
in a file with variable length lines where we cache common
descent nodes to minimize redundant file seeks in a service.
This is similar to code in my DEM waterflow analyzer and
was ported from python to go.
Some ideas borrowed from bibliographic indexing along with rway merge.
Seeing if go can come close to the native linux sort. 

Note: We want the merge to run as fast as possible consuming
all available cores but it is generally lower in priority
than the actual search actvity so we assume we can use all
reported cores then use proces priority to ensure the online
traffic gets the CPU when it is needed. 

General rule for latter when running a continous indexing system
we always want to merge small files as a group before merging larger
files to minimize total IO traffice. */

type mergeSpec struct {
	fnames []string
	fnameOut string
}

type SortLines struct {
	startFiSet []string // list of files to work on.
	fiCtr      int      // used to generate new file names
	baseName   string
	wrkBaseName string
	suff       string
	workList   chan mergeSpec
	fiList     chan string
	wgDone     sync.WaitGroup
	wgWorkerReq sync.WaitGroup
	pendCnt    int // count of files known to need work
	countLock  sync.Mutex
	maxThread  int
	targFilesPerThread int
	activeThreads int
}

const MaxOpenFiles = 400

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	return err != nil
}
func fileSize(path string) int64 {
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Println("L68: Error fileSize() path=", path, " err=", err)
		return -1
	}
	return fi.Size()
}

func (sln *SortLines) nextFiName() string {
	sln.countLock.Lock()
	sln.fiCtr += 1
	sln.countLock.Unlock()
	return sln.wrkBaseName + "-" + fmt.Sprintf(".merge.s-%03d", sln.fiCtr) + sln.suff
}

/* Remove source file if it matches the defined file suffix
  returns true if the file was removed otherwise returns 
  false */
func (sln *SortLines) removeTempFile(fiPath string) bool{
	ext := filepath.Ext(fiPath)
	if ext == sln.suff {
		err := os.Remove(fiPath)
		if err != nil {
			fmt.Println("L76: Err Removing file: ", fiPath, " err=", err)
			return false
		}
		return true
	} else {
		return false
	}
}

func  (sln *SortLines) mergeFileSet(mreq mergeSpec) {
	fout, foerr := os.OpenFile(mreq.fnameOut, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
		log.Fatalf("failed creating file %s: %s", mreq.fnameOut, foerr)
	}
	fnames := mreq.fnames
	datawriter := bufio.NewWriter(fout)
	
	numFile := len(fnames)
	fmt.Println("L132: initial fileList len=", numFile," files=", fnames)
	files := make([]*os.File, numFile)
	scanners := make([]*bufio.Scanner, numFile)
	lines := make([]string, numFile)
	// initialize line buffers so we do not need to re-allocate
	//for ndx:=0; ndx < numFile; ndx++ {
	//	lines[ndx] = make([]byte,6400)
	//}
	//lines = make([]string, numFile)
	// Open our Array of Files 
	for ndx:= 0; ndx < numFile; ndx++ {
		fiPtr, err := os.Open(fnames[ndx])
		if err != nil {
			log.Fatalf("failed opening file %s: %s", fnames[ndx], err)
			return
		}
		files[ndx] = fiPtr
		scanner := bufio.NewScanner(fiPtr)
		scanners[ndx] = scanner
	}
	fmt.Println("L144: All files open")
	bytesRead := 0
	linesRead := 0
	
	// Read a starter line from every file
	for ndx:= 0; ndx < numFile; ndx++ {
	 	if files[ndx] == nil {
			// this file has reached EOF so skip
			fmt.Println("L155: Skip file closed")
			//lines[ndx] = nil
			lines[ndx] = "~"
			continue
		}
		more := scanners[ndx].Scan()
		str1 := scanners[ndx].Text()
		blen := len(str1)
		bytesRead += blen
		linesRead += 1
		lines[ndx] = str1
		if more == false {
			files[ndx]=nil
			scanners[ndx]=nil
		}
	}
	
	
	for {
		//fmt.Println("Outer Loop linesRead=", linesRead)
		lowest := lines[0]
		lowestNdx := 0
		// Find the file with the lowest sort sequence
		for ndx:=0; ndx < (numFile); ndx++ {			
			if lines[ndx] == "~" {
				// Clear files so we can skip them
				files[ndx] = nil
				continue;
			}
			if lowest == "~" {
				// Use this item no matter what since the
				// prior one is no good.
				lowest = lines[ndx];
				lowestNdx = ndx;
				continue
			}
			
			if lowest > lines[ndx] {
				// Set a new lowest fp
				lowest = lines[ndx];
				lowestNdx = ndx;
				continue;
			} 
		}
		
		if lowest == "~" {
			// last line for all files must be nil which indicates
			// EOF has been reached for all files
			break;
		}
		
		// Based on the Lowest Identified Write that string 
		// to disk
		datawriter.WriteString(lowest + "\n")
		if files[lowestNdx] == nil {
			// no more to read from this file
			lines[lowestNdx] = "~"
			continue;
		} 
		
		more := scanners[lowestNdx].Scan()
		str1 := scanners[lowestNdx].Text()
		if more == false {
			files[lowestNdx]=nil
			scanners[lowestNdx]=nil
		}
		lines[lowestNdx] = str1
		blen := len(str1)
		bytesRead += blen
		linesRead += 1
	} // for merge loop
	
	datawriter.Flush()
	fout.Close()
	// Cleanup and update our status in larger queue
	for ndx := 0; ndx < len(fnames); ndx ++ {
		fmt.Println("L218: Cleanup remove ", fnames[ndx])
		files[ndx].Close()
		sln.removeTempFile(fnames[ndx])
		sln.wgDone.Done()
	}
	
	// Update Status of greater SLN merge so it knows
	// this work has been completed.
	fmt.Println("L225: Finished mergeSet mReq bytesRead=", bytesRead, " linesRead=", linesRead, "len fnames=", len(fnames), " fnameout=", mreq.fnameOut)
	sln.fiList <- mreq.fnameOut // Add output file to be re-processed
	sln.countLock.Lock()
	sln.pendCnt -= numFile
	sln.activeThreads --
	sln.countLock.Unlock()
	fmt.Println("L230: finished mergeSet pendCnt=", sln.pendCnt, " len sln.fiList=", len(sln.fiList), " wrkListLen=", len(sln.workList), " fnameout=", mreq.fnameOut)
	sln.wgWorkerReq.Done()
}


var emptyMergeReq mergeSpec
// My worker thread loops looking for work
// until the channel is closed
func (sln *SortLines) fpWorkThread() {
	for {
		mreq, more := <-sln.workList // dequeue
		if len(mreq.fnames) == 0 {
			// channel has been closed
			break;
		}
		sln.wgWorkerReq.Add(1)
		sln.countLock.Lock()
		sln.activeThreads ++
		sln.countLock.Unlock()
		fmt.Println("L243: Worker pendCnt=", sln.pendCnt, " sln.activeThreads=", sln.activeThreads, " mreq=", mreq, " more=", more)
		sln.mergeFileSet(mreq)
		if more == false {
			break
		}
	}
}

/* Merge a group of files and return return the name of the output file produced.
The files are grouped into clusters of targFilesPerThread then enqueued so they
are available for the worker threads. As output files are produced they may 
need to be mereged with other files so we enqueue it into the working set.  
This continues until we work our way down to a single output file which is when
we know that we are done. As we work through the initial a probable condition is
that there not enough files left to merge and meet the targFilesPerThread count
but will still need to process them. The goal is produce the desired single
merged file in the smallest number of merge phases possible while leveraging
parralellism early in the process to maximise usage of IO device bandwidth. 
*/
func (sln *SortLines) mergFilesList() string {
	fmt.Println("L262: sln=", sln)
	fmt.Println("L263: launch Threads maxThread=", sln.maxThread)
	// Spawn our worker threads passing in SLN so they can
	// process the queued work items.
	for threadCnt := 0; threadCnt <= sln.maxThread; threadCnt++ {
		go sln.fpWorkThread()
		fmt.Println("L268: Lauched Thread # ", threadCnt)
	}

	// Load the sln.workList Queue with items to work on.
	waitCnt := 0
	nextBatchList := make([]string,0)
	batchList := make([]string,0)
	for sln.pendCnt > 1 {
		fmt.Println("L275: pendCnt=", sln.pendCnt, "len batchList=", len(batchList), " lenWorkList=", len(sln.workList), " lenFiList=", len(sln.fiList))
		if len(sln.fiList) <= 1 && sln.pendCnt > len(batchList)+1{
			// May be waiting for an existing merge to be completed
			// so a file could still show up to be merged.  
			waitCnt++
			fmt.Println("L280:merge thread waitCnt=", waitCnt,  " pendCnt=", sln.pendCnt, " sln.activeThreads=", sln.activeThreads, " lenWrkList=", len(sln.workList))
			time.Sleep(time.Second)
			continue;
		}
		waitCnt = 0
		doMerge := false
		
		fname, _ := <-sln.fiList // Get next file to be processed.
		
		// Special Case Allow merge if prior set if the size of the file read is
		// more than X larger than the last file added to the working set. It is
		// faster to merge the smaller files first then merge a smaller number 
		// of larger files to minimize compare overhead. This is overrulled if
		// when the fileis small enough even though it is larger because it will
		// still be faster than possibly running an extra phase. As the files
		// get larger this rule becomes critical to preserve performance.
		if len(batchList) > 1 {
			fsize := fileSize(fname)
			fsizeLast := fileSize(batchList[len(batchList)-1])
			// Add logic to skip special case if we could finish
			// in this pass. 
			if fsize > int64(float64(fsizeLast) * 1.95) {
				doMerge = true
				nextBatchList = append(nextBatchList, fname)
				fname = "~SKIP" // ignore this file for this bath step
			}
		}
		
		if fname > "" && fname != "~SKIP" { 
			batchList = append(batchList, fname)
		}
		
		nwrk := len(batchList)
		if fname == "" {
				// Flush because the channel has been closed
				// Should only occur during exceptional abort
				// or error conditions.
				doMerge = true
				sln.countLock.Lock()
				sln.pendCnt = 0 // trigger break in next ieteration
				sln.countLock.Unlock()
		}
		if nwrk >= sln.targFilesPerThread { doMerge = true} //  Flush because we have enough files. 
		if nwrk >= (sln.pendCnt) { doMerge = true } // Flush because we have nearly completed the merge this will be the last pass.
		if sln.pendCnt <= 2 && nwrk >= 2 { doMerge = true } // This is our final merge // fall back may never be reached
		
		if doMerge {
			sln.countLock.Lock()
			sln.pendCnt++ // adjust for output file
			sln.fiCtr++
			sln.countLock.Unlock()
			foutName := sln.nextFiName()
			fmt.Println("L323: pendCnt=", sln.pendCnt, " len batchList=", nwrk, " foutName=", foutName, " batchList=", batchList)
			sln.wgDone.Add(1)
			workReq := mergeSpec{fnames: batchList, fnameOut: foutName}
			sln.workList <- workReq // enqueu
			fmt.Println("L327: Queued=", sln.pendCnt, " lenFiList=", len(sln.fiList), " lenWorkList=", len(sln.workList), " workReq=", workReq)
			batchList =  nextBatchList
			nextBatchList = make([]string,0)
		}
		if fname == "" { break} // empty string means channel close to no more work could arrive.
	} // main merge loop
	
	sln.wgDone.Done() // for the last file that does not need to be processed.
	fmt.Println("L334: pendCnt=", sln.pendCnt, " Waiting for Merge to finish", " lenFiList=", len(sln.fiList), " lenWorkList=", len(sln.workList))
	// TODO: Add files written to the finish list as soon as the merge
	//  above finishes so we can start on next merge as soon as we have
	//  processor resources available.``
	close(sln.workList) // tell our worker threads they can close
	close(sln.fiList)
	sln.wgDone.Wait()
	sln.wgWorkerReq.Wait()
	outFiName := <- sln.fiList // last remaining file shoud be our output
	fmt.Println("L343: All worker Threads are finished outFi=", outFiName)
	return outFiName
}

func main() {
	portCoreToUse := 1.7
	numMergeThreads := int(float64(runtime.NumCPU()) * portCoreToUse)  // reset starting default for current machine config
	// May need to reduce maxThreads to avoid
	// exceeding machine ulimits for files open
	maxThreadFileHand := MaxOpenFiles / 3
	if numMergeThreads > maxThreadFileHand {
		numMergeThreads = maxThreadFileHand
	}
	if numMergeThreads < 1 {
		numMergeThreads = 1
	}
	
	fmt.Println("toSysCore=", runtime.NumCPU(), " portCoreToUse=", portCoreToUse, " numMergeThreads=", numMergeThreads)

	suffix := "seg"
	fmt.Println("os.Args=", os.Args)
	if len(os.Args) < 4 {
		fmt.Println("Arg 1 input base name, arg2 = work Base Name for Temp files, arg3= output base name")
		return 
	}

	baseName := os.Args[1]
	wrkBaseName := os.Args[2]
	fOutName := os.Args[3]
	fmt.Println("baseName=", baseName)

	if fileExists(fOutName) {
		err := os.Remove(fOutName)
		if err != nil {
			fmt.Println("ERROR could not remove old output file ", fOutName, " err=", err)
		}
	}

	globPat := baseName + "*-" + fmt.Sprintf("%03d", 0) + "." + suffix
	fmt.Println("L161: globPat=", globPat)
	// Get a list of files
	fileList, _ := filepath.Glob(globPat)
	if fileList == nil {
		fmt.Println("L322 Found no files for globPat=", globPat)
	}
	fmt.Println("L132: initial fileList len=", len(fileList))
	numFi := len(fileList)
	// Start with 15 file per thread but large core systems we can reduce this so
	// we have each core working on a smaller set which reduces the compare
	// overhead.  Our goal is to minimize re-processing files which encourages a
	// larger number but this slows things down in the compare loop so we never
	// want to exceed a maximum threashold. On other hand merging large files
	// again latter so minimizing additional phases is also important.
	// If we have more than we can reasonably merge in a single pass without
	// compare overhead getting too high we want to spread the work across
	// as many cores as possible since they can all run simutaneously. 
	maxFilesPerThead := 7 
		// Tuned so when in early process with lots of segments to 
		// to merge, we hit Max Read speed from SATA SSD when CPU is at
		// 85%.   When CPU are faster or we have more cores relative to
		// drive speed then we can increase this number to reduce passes
		// when we have faster drives relative to avaialble cores then
		// reduce this number. 
	minFilesPerThread := 4
	filesPerThread := (numFi / numMergeThreads) + 1
	if filesPerThread > maxFilesPerThead {
		filesPerThread = maxFilesPerThead
	}
	if filesPerThread < minFilesPerThread {
		filesPerThread = minFilesPerThread
	}
	fmt.Println("L360 filesPerThread=", filesPerThread, " numFi=", numFi,  " numThread=", numMergeThreads)


	sln := &SortLines{
		startFiSet: fileList,
		fiCtr:      0,
		baseName:   baseName,
		wrkBaseName: wrkBaseName,
		suff:       ".srt",
		workList:   make(chan mergeSpec, 2000),
		fiList:     make(chan string, 2000),
		maxThread:  numMergeThreads,
		targFilesPerThread : filesPerThread }
	
	// Add the files to work to a channel
	for ndx := 0; ndx < len(fileList); ndx++ {
		sln.fiList <- fileList[ndx] // enqueue
		sln.wgDone.Add(1)
		sln.pendCnt++
	}
	lastFile := sln.mergFilesList()
	fmt.Println("L283: lastFile=", lastFile)
	fmt.Println("L284: finished waiting on threads start next phase")
	fmt.Println("L285: Renaming ", lastFile, " to ", fOutName)
	err := os.Rename(lastFile, fOutName)
	if err != nil {
		fmt.Println(" Error renaming file segment from ", fileList[0], " to ", fOutName, " err=", err)
	}
}
