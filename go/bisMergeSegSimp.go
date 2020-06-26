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
	//"time"
)

/* Merge pre-sorted input files into a sorted output file.   This technique is based roughly
on the bisect file pattern used in my DEM waterflow array ported from python to go.
Some ideas borrowed from bibliographic indexing along with rway merge.
Seeing if go can come close to the native linux sort.

Adapted from Nieve approach the uses a smaller number of threads to read
a larger number of files using a sort to one that uses a larger number
of threads but but avoids the need for any sort activities.
Written to test idea that if we can reduce two file merges to max
speed that we can better afford to re-read the files multiple
times. 	*/

type mergeSpec struct {
	fin1 string
	fin2 string
	fout string
}

type SortLines struct {
	startFiSet []string // list of files to work on.
	fiCtr      int      // used to generate new file names
	baseName   string
	suff       string
	workList   chan mergeSpec
	fiList     chan string
	wgDone     sync.WaitGroup
	pendCnt    int // count of files known to need work
	countLock  sync.Mutex
	maxThread  int
}

const MaxOpenFiles = 300

var MaxMergeThreadsActive = 50

func fileExists(name string) bool {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	return err != nil
}

func (sln *SortLines) nextFiName() string {
	sln.countLock.Lock()
	sln.fiCtr += 1
	sln.countLock.Unlock()
	return sln.baseName + "-" + fmt.Sprintf("s-%03d", sln.fiCtr) + "." + sln.suff
}

/* Read strings from a set of files concurrently open
find the read lines from both files take the lowest
sort order string and read next line.  Keep going
until we run out.  */
func (sln *SortLines) mergeFiles(fname1 string, fname2 string, fnameOut string) {

	fmt.Println("L74: fnameOut=", fnameOut, " fname1=", fname1, " fname2=", fname2)

	fiPtr1, err1 := os.Open(fname1)
	if err1 != nil {
		log.Fatalf("failed opening file %s: %s", fname1, err1)
	}

	fiPtr2, err2 := os.Open(fname2)
	if err2 != nil {
		log.Fatalf("failed opening file %s: %s", fname2, err2)
	}

	scan1 := bufio.NewScanner(fiPtr1)
	scan2 := bufio.NewScanner(fiPtr1)
	more1 := scan1.Scan()
	more2 := scan2.Scan()
	text1 := scan1.Text()
	text2 := scan2.Text()
	defer fiPtr1.Close()
	defer fiPtr2.Close()
	lastWrite := ""
	fout, foerr := os.OpenFile(fnameOut, os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
		log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()

	// Main Read & Compare Loop
	for more1 == true && more2 == true {
		if text1 < text2 {
			if text1 != lastWrite {
				fout.WriteString(text1)
				fout.WriteString("\n")
				lastWrite = text1
			}
			more1 = scan1.Scan()
			text1 = scan1.Text()
		} else if text2 < text1 {
			if text2 != lastWrite {
				fout.WriteString(text2)
				fout.WriteString("\n")
				lastWrite = text2
			}
			more2 = scan2.Scan()
			text2 = scan2.Text()
		} else {
			if text1 != lastWrite {
				fout.WriteString(text1)
				fout.WriteString("\n")
				lastWrite = text1
			}
			more1 = scan1.Scan()
			text1 = scan1.Text()
			more2 = scan2.Scan()
			text2 = scan2.Text()
		}
	} // for main read compare

	// Finish file 1 since it may have some records left
	// over when file 2 runs out
	for more1 == true {
		if text1 != lastWrite {
			fout.WriteString(text1)
			fout.WriteString("\n")
			lastWrite = text1
		}
		more1 = scan1.Scan()
		text1 = scan1.Text()
	}

	// Finish file 2 since it may have some records left
	// over when file 1 runs out
	for more2 == true {
		if text2 != lastWrite {
			fout.WriteString(text2)
			fout.WriteString("\n")
			lastWrite = text2
		}
		more2 = scan1.Scan()
		text2 = scan1.Text()
	}

	// TODO:remove the input files files.

	datawriter.Flush()
	sln.countLock.Lock()
	sln.pendCnt -= 2 // Finish 2 files
	sln.countLock.Unlock()
	sln.fiList <- fnameOut
	sln.wgDone.Done() // fin1 done
	sln.wgDone.Done() // fin2 done
}

// My worker thread loops looking for work
// until the channel is closed
func (sln *SortLines) fpWorkThread() {
	for {
		mreq, more := <-sln.workList // dequeue
		sln.mergeFiles(mreq.fin1, mreq.fin2, mreq.fout)
		if more == false {
			break
		}
	}
}

/* Merge a group of files and return the list of
of files produced.  Works throught the list of input
files from a channel 2 at a time. When it produces
an output file that file is added back to the channel
to be processed.
*/
func (sln *SortLines) mergFilesList() []string {
	foutCnt := 0
	fmt.Println("L160 sln=", sln)
	filesWritten := make([]string, 0)
	batchCnt := 0

	// Spawn our worker threads passing in SLN so they can
	// read the work items queue.
	for threadCnt := 0; threadCnt <= sln.maxThread; threadCnt++ {
		go sln.fpWorkThread()
	}

	// Now Load the Queue with items to work on.
	for sln.pendCnt > 1 {
		// This is a little complex so some explanation is in order
		// We start with pendCnt equal to number of input items in
		// the list.  Whenever we merge 2 we output 1 which may need
		// to be merged with something else giving us a net reduction
		// of 1.  When we work down to having only 1 item in the queue
		// then we know that everything has been merged as far as possible.
		// so we can exit.
		f1Name, _ := <-sln.fiList // dequeue
		if f1Name == "" {
			break
		}

		f2Name, _ := <-sln.fiList // dequeue
		if f2Name == "" {
			break
		}
		batchCnt++
		foutCnt++
		foutName := sln.nextFiName()
		fmt.Println("L173: MergeSet=", sln.startFiSet, " foutName=", foutName)
		filesWritten = append(filesWritten, foutName)
		sln.countLock.Lock()
		// adusting counters that will be modified by
		// by working threads.
		sln.pendCnt++ // adjust for output file
		sln.wgDone.Add(1)
		sln.countLock.Unlock()

		// Enque the actual work for two input files to be
		// picked up by one of our worker threads.  Since we
		// have a limit on buffer length this may block until
		// other threads finish.
		workReq := mergeSpec{fin1: f1Name, fin2: f2Name, fout: foutName}
		sln.workList <- workReq // enqueu

	} // main merge loop
	sln.wgDone.Done() // for the last file that does not need to be processed.
	fmt.Println("L284 Waiting for Merge to finish")
	// TODO: Add files written to the finish list as soon as the merge
	//  above finishes so we can start on next merge as soon as we have
	//  processor resources available.``
	sln.wgDone.Wait()
	fmt.Println("L294: All worker Threads are finished =", filesWritten)
	close(sln.workList) // tell our worker threads they can close
	close(sln.fiList)
	return filesWritten
}

func main() {
	portCoreToUse := 10.0
	numCore := int(float64(runtime.NumCPU()) * portCoreToUse)
	if numCore < 2 {
		numCore = 2
	}
	MaxMergeThreadsActive = numCore + 1 // reset starting default for current machine config
	// May need to reduce maxThreads to avoid
	// exceeding machine ulimits for files open
	maxThreadFileHand := MaxOpenFiles / 3
	if MaxMergeThreadsActive > maxThreadFileHand {
		MaxMergeThreadsActive = maxThreadFileHand
	}

	suffix := "seg"
	fmt.Println("os.Args=", os.Args)
	if len(os.Args) < 3 {
		fmt.Println("Arg 1 must be input file name and arg2 must be output base name")
		panic("abort")
	}

	baseName := os.Args[1]
	fOutName := os.Args[2]
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
	fmt.Println("L132: initial fileList len=", len(fileList), " files=", fileList)

	sln := &SortLines{
		startFiSet: fileList,
		fiCtr:      0,
		baseName:   baseName,
		suff:       "srt",
		workList:   make(chan mergeSpec, 2000),
		fiList:     make(chan string, 2000),
		maxThread:  maxThreadFileHand}
	// Add the files to work to a channel
	for ndx := 0; ndx < len(fileList); ndx++ {
		sln.fiList <- fileList[ndx] // enqueue
		sln.wgDone.Add(1)
		sln.pendCnt++
	}
	filesWritten := sln.mergFilesList()
	fmt.Println("L283: filesWritten=", filesWritten)
	fmt.Println("L284: finished waiting on threads start next phase")
	fmt.Println("L285: Renaming ", fileList[0], " to ", fOutName)
	err := os.Rename(fileList[0], fOutName)
	if err != nil {
		fmt.Println(" Error renaming file segment from ", fileList[0], " to ", fOutName, " err=", err)
	}
}
