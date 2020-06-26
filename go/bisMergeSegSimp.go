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
	"sync"
	"runtime"
	"time"
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

type SortLines struct {
	fiSet []string // list of files to work on.
	fiCtr int // used to generate new file names
	baseName string
	suff string
	workCnt int // count of known items remaining to process
}

const MaxOpenFiles = 300
var MaxMergeThreadsActive = 50
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

func (sln *SortLines) nextFiName() string {
	countLock.Lock()
	sln.fiCtr += 1
	countLock.Unlock()
	return sln.baseName + "-" + fmt.Sprintf("s-%03d",sln.fiCtr) + "." + sln.suff
}

/* Read strings from a set of files concurrently open
find the read lines from both files take the lowest
sort order string and read next line.  Keep going 
until we run out.  */
func (sln *SortLines) mergeFiles(fname1 string, fname2 string, fnameOut string) {
	fmt.Println("L204: fnameOut=", fnameOut, " fname1=", fname1, " fname2=", fname2);
	
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
	lineCnt := 0
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
			more1 := scan1.Scan()
			text1 := scan1.Text()
		} else if text2 < text1 {
			if text2 != lastWrite {
				fout.WriteString(text2)
				fout.WriteString("\n")
				lastWrite = text2
			}
			more2 := scan2.Scan()
			text2 := scan2.Text()
		} else {
			if text1 != lastWrite {
				fout.WriteString(text1)
				fout.WriteString("\n")
				lastWrite = text1
			}
			more1 := scan1.Scan()
			text1 := scan1.Text()
			more2 := scan2.Scan()
			text2 := scan2.Text()
		}
	} // for main read compare
	
	// TODO: Finish file 1
	
	// TODO Finsih file 2 

	// TODO:remove the segment files. 
	
	datawriter.Flush()
	countLock.Lock()
		openFileCnt -= 3
		mergeThreadsActive -= 1
		sln.fiSet = append(sln.fiSet, fnameOut)
		sln.workCnt -= 2 // represent the fact that we have finished merging two input files
	countLock.Unlock()
	procSortLinesWG.Done()
}





/* Merge a group of files and return the list of 
  of files produced.  Breaks the input set up based
  on the maximum number of files specified. 
  */
func (sln *SortLines) mergFilesList() []string {
	foutCnt := 0
	if len(sln.fiSet) <= 1 {
		return sln.fiSet
	}
	fmt.Println("L160 sln=", sln)
	filesWritten := make([]string,0)
	var fnames = make([]string,0)
	batchCnt := 0
	for ndx:= 0; ndx<len(sln.fiSet) && len(sln.fiSet) > 1; ndx+= 2 {
		batchCnt += 1
		foutCnt+= 1
		
		foutName := sln.nextFiName()
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
		f1Name := sln.fiSet[ndx]
		f2Name := sln.fiSet[ndx + 1]
		countLock.Lock()
			// adusting counters that will be modified by
			// by working threads. 
			openFileCnt += 3
			mergeThreadsActive += 1
			sln.fiSet = sln.fiSet[2:] // remove them since we are already working on them
			sln.workCnt += 1 // to represent the need to potentially merge the output file.
		countLock.Unlock()
		
		// Start the GO Routine / Thread to do the actual processing
		go sln.mergeFiles(f1Name, f2Name, foutName)
		for sln.workCnt > 1 && len(sln.fiSet) < 2 {
			// Can have condition of no more work to do yet because
			// we are waiting for other mergest to finish but have
			// consumed all files in this phase so we need to wait
			// TODO: Conver this to the GORoutine Channel mechanism
			//  to make the logic a little more clear
			fmt.Println("L200: Waiting for more work workCnt=", workCnt, "#files=", len(flist))
			time.Sleep(time.Second)
		}
	} // main merge loop
	
	fmt.Println("L284 Waiting for Merge to finish" )
	// TODO: Add files written to the finish list as soon as the merge 
	//  above finishes so we can start on next merge as soon as we have
	//  processor resources available. 
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
	sln := &SortLines { 
			fiSet : fileList,
			fiCntr : 0,
			baseName: baseName,
			suff: "srt",
			workCnt: len(fileList) }
	filesWritten = sln.mergFilesList()
	fmt.Println("L294: Waiting on Threads for files=", fileList)
	procSortLinesWG.Wait()
	fmt.Println("L331: finished waiting on threads start next phase")
	fmt.Println("L336: Renaming ", fileList[0], " to ", fOutName)
	err := os.Rename(fileList[0], fOutName)
	if err != nil {
	   fmt.Println(" Error renaming file segment from ", fileList[0], " to ", fOutName, " err=", err)
	}
}