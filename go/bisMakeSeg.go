package main
 
import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sort"
	//"strconv"
	"runtime"
	"sync"
)

/* Parse the input file in the form of 
package view_name, view_oid, source_name, source_oid 
convert it to source_oid, source_oid, source_name, view_name
to allow access using binary search techniques.   This technique
is based roughly on the bisect file pattern used in my DEM water
flow array ported from python to go.  Some ideas borrowed from
bibliographic indexing along with rway merge. Seeing if go
can come close to the native linux sort. 

The line parser turned out to be a source of load that 
was limiting the rate at which we could feed the worker
tasks so so moved that function to the sort threads 
so we could spread it acorss multiple-cpu.

*/

type BlockDesc struct {
  segCnt int
  baseFiName string
  lines []string
}


//------------------
//--- Process Input Files
//------------------
func saveBlock(bdesc *BlockDesc) {
	fiName := bdesc.baseFiName + "." + fmt.Sprintf("%05d",bdesc.segCnt) + "-000.seg"
	file, err := os.OpenFile(fiName, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
		  log.Fatalf("failed creating file %s: %s", fiName, err)
	}
	datawriter := bufio.NewWriter(file)
	defer file.Close()
	numRec := len(bdesc.lines)
	for i:=0; i<numRec;i++ {
	  _, _ = datawriter.WriteString(bdesc.lines[i] + "\n")
	}
	datawriter.Flush()
	//PrintMemUsage()
}

func procBlock(chanIn chan *BlockDesc, wg *sync.WaitGroup) {
	for {
	    bdesc,more := <- chanIn
		if bdesc == nil {
			break // Channel has been closed
		}
		fmt.Println("proc Block rec lines segCnt=", bdesc.segCnt, " numLines=", len(bdesc.lines))
		numLine := len(bdesc.lines)
		// Perform string / line conversion in worker thread
		// to minimize load in block loader
		var lflds [4] string
		for ndx:=0; ndx < numLine; ndx++ {
			larr := strings.Split(bdesc.lines[ndx], ",")
			//fmt.Println("larr=", larr)
			if len(larr) != 4  {
				bdesc.lines[ndx] = "~ERROR" + bdesc.lines[ndx]
				continue
			} else {
				lflds[0] = larr[3]
				lflds[1] = larr[1]
				lflds[2] = larr[2]
				lflds[3] = larr[0]
				//fmt.Println("lfds=", lflds)
				bdesc.lines[ndx] = strings.Join(lflds[:], ",")
			}
		}
		sort.Strings(bdesc.lines)
		saveBlock(bdesc)
		if (more) { 
			fmt.Println("proc Block more")
		} else {
			fmt.Println("proc Block no more")
			break
		}
	}
	wg.Done()
}

func PrintMemUsage() {
        var m runtime.MemStats
        runtime.ReadMemStats(&m)
        // For info on each, see: https://golang.org/pkg/runtime/#MemStats
        fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
        fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
        fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
        fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func bToMb(b uint64) uint64 {
    return b / 1024 / 1024
}

func main() {
	fractOfCPUUsage := 1.4
	numThread := int(float64(runtime.NumCPU()) * fractOfCPUUsage)
	if numThread < 2 { numThread = 2 }
	fmt.Println("selected max thread=", numThread)
	var m runtime.MemStats
    runtime.ReadMemStats(&m)
	systemBytes := m.Sys * 1024
	expectedBytesPerLineAvg := 130
	portOfRamToUse := 0.070
	maxBytesPerBlock := int((float64(systemBytes) / float64(numThread +2)) * float64(portOfRamToUse))
	maxElePerBlock := maxBytesPerBlock / expectedBytesPerLineAvg
	numBlockDesc := 2
	if numBlockDesc < 1 {
		numBlockDesc = 1
	}
	var wg sync.WaitGroup
	wg.Add(numThread)
	blocks := make(chan *BlockDesc, numBlockDesc)
	fmt.Println("os.Args=", os.Args)
	if len(os.Args) < 3 {
		fmt.Println("Arg 1 must be input file name and arg2 must be output base name")
		panic("abort")
	}
	fInName := os.Args[1]
	fOutName := os.Args[2]
	fmt.Println("fiName=", fInName)
	file, err := os.Open(fInName)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	
	// Spawn our worker threads
	for pcnt:=0; pcnt<numThread; pcnt++ { 
		go procBlock(blocks, &wg);		
	}
	
	// -----
	// -- Build our buffers
	// -----
	aline := "" 
	scanner := bufio.NewScanner(file)
	//scanner.Split(bufio.ScanLines)
	scanner.Scan()
	header :=  scanner.Text()
	fmt.Println("header=", header)
	txtlines := make([]string, maxElePerBlock+1)
	
	rowNdx := 0
	buffBytes := 0
	segCnt := 0
	lineCnt := 0
	// TODO:  This thread seems to be blocking the feed
	//  for other threads.  Consider moving work out 
	//  allowing multiple readers on separate threads
	// 
	for scanner.Scan() {
		aline = scanner.Text()
		if aline > " " {
			lineCnt += 1
			
			txtlines[rowNdx] = aline
			  rowNdx += 1
			//  buffBytes += len(outStr)
			  buffBytes += len(aline)
			  if buffBytes >= maxBytesPerBlock || rowNdx >= maxElePerBlock {
				  // Flush our full buffer
				  fmt.Println("Buffer line#", lineCnt)
				  blocks <- &BlockDesc{segCnt, fOutName, txtlines[:rowNdx]}
				  // make a new buffer to contain the next chunk
				  // so we can fill it while other threads work
				  // on sorting
				  txtlines = make([]string,maxElePerBlock)
				  rowNdx = 0
				  buffBytes = 0
				  segCnt += 1
			 // }
			}
		}
	}
	// flush the last buffer full
	if rowNdx > 0  {
		blocks <- &BlockDesc{segCnt, "testseg", txtlines[:rowNdx]}
	}
	file.Close()
	close(blocks)
	fmt.Println("Waiting for threads to finish")
	wg.Wait()
	
	//for _, eachline := range txtlines {
	//	fmt.Println(eachline)
	//}
	
}
 