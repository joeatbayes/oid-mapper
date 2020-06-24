package main
 
import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"sort"
	//"strconv"
)

/* Parse the input file in the form of 
package view_name, view_oid, source_name, source_oid 
convert it to source_oid, source_oid, source_name, view_name
to allow access using binary search techniques.   This technique
is based roughly on the bisect file pattern used in my DEM water
flow array ported from python to go.  Some ideas borrowed from
bibliographic indexing along with rway merge. Seeing if go
can come close to the native linux sort. */

type BlockDesc struct {
  segCnt int
  baseFiName string
  lines []string
}


//------------------
//--- Process Input Files
//------------------
func saveBlock(bdesc BlockDesc) {
	fiName := bdesc.baseFiName + "." + fmt.Sprintf("%05d",bdesc.segCnt) + ".seg"
	file, err := os.OpenFile(fiName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
}

func procBlock(chanIn chan BlockDesc) {
	for {
	    bdesc,more := <- chanIn
		fmt.Println("proc Block rec lines segCnt=", bdesc.segCnt, " numLines=", len(bdesc.lines))
		sort.Strings(bdesc.lines)
		saveBlock(bdesc)		
		if (more) { 
			fmt.Println("proc Block more")
		} else {
			fmt.Println("proc Block no more")
			break;
		}
	}
}



func main() {
	numThread := 25
	blocks := make(chan BlockDesc, numThread+1)
	fmt.Println("os.Args=", os.Args)
	if len(os.Args) < 2 {
		fmt.Println("Arg 1 must be input file name")
		panic("abort")
	}
	fInName := os.Args[1]
	fmt.Println("fiName=", fInName)
	file, err := os.Open(fInName)
	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	
	// Spawn our worker threads
	for pcnt:=0; pcnt<numThread; pcnt++ { 
		go procBlock(blocks);		
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
	txtlines := make([]string, 2000000)
	var lflds [4] string
	rowNdx := 0
	buffBytes := 0
	segCnt := 0
	lineCnt := 0
	for scanner.Scan() {		
		aline = scanner.Text()
		if aline > " " {
			lineCnt += 1
			larr := strings.Split(aline, ",")
			//fmt.Println("larr=", larr)
			if len(larr) == 4  {
			  lflds[0] = larr[3]
			  lflds[1] = larr[1]
			  lflds[2] = larr[2]
			  lflds[3] = larr[0]
			  //fmt.Println("lfds=", lflds)
			  outStr := strings.Join(lflds[:], ",")
			  txtlines[rowNdx] = outStr
			  rowNdx += 1
			  buffBytes += len(outStr)
			  if buffBytes >= 150000000 || rowNdx > 1999999 {
				  // Flush our full buffer
				  fmt.Println("Buffer line#", lineCnt)
				  blocks <- BlockDesc{segCnt, "testseg", txtlines[:rowNdx]}
				  // make a new buffer to contain the next chunk
				  // so we can fill it while other threads work
				  // on sorting
				  txtlines = make([]string,2000000)
				  rowNdx = 0
				  buffBytes = 0
				  segCnt += 1
			  }
			}
		}
	}
	// flush the last buffer full
	if rowNdx > 0  {
		blocks <- BlockDesc{segCnt, "testseg", txtlines[:rowNdx]}
	}
								  
 
	file.Close()
	close(blocks)
	for _, eachline := range txtlines {
		fmt.Println(eachline)
	}
	
}
 