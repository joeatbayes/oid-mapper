
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
to allow access using binary search techniques */

type SortLine struct {
	lastString string
	fp *os.File 
	scanner *bufio.Scanner
	eof bool
}

type SortLines []SortLine

type BlockDesc struct {
  segCnt int
  baseFiName string
  lines []string
}


// open all the files names encapsulate them
// in a sort spec and return a slice with all 
// the specs.
func makeSortLines(fnamesIn []string) SortLines {
	specs := make([]SortLine, len(fnamesIn))
	for ndx:=0; ndx < len(fnamesIn); ndx++ {
		fiName := fnamesIn[ndx]
		fiPtr, err := os.Open(fiName)
		if err != nil {
			log.Fatalf("failed opening file %s: %s", fiName, err)
		} else {
			scanner := bufio.NewScanner(fiPtr)
			aspec := SortLine {"", fiPtr, scanner, false}
			specs = append(specs, aspec);
		}
	}
	return specs
}

func (sl SortLines)sortSortLines() {
   sort.Slice(sl, func(i, j int) bool { return sl[i].lastString < sl[j].lastString })
}

func (sel SortLine) readNext() string {
	if sel.eof == true {
	  sel.lastString = "~~"
	} else {
		more := sel.scanner.Scan()
		if more == false {
			sel.eof = true
		} 
		sel.lastString = sel.scanner.Text()
	}
	return sel.lastString
}

/* Read the next line for all files in the sort line spec */
func (sln SortLines) readNextAll() SortLines {
	for ndx:=0; ndx < len(sln); ndx ++ {
		sln[ndx].readNext();
	}
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
	
	fout, foerr := os.OpenFile(fnameOut, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
			log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	for {
		if len(sln) == 0 {
			// all files have been removed from consideration
			break
		}
		if selected.lastString > next.lastString {
			sln.sortSortLines()
			selected = sln[0]
			next = sln[1]
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
			if len(sln) == 1 {
				break
			}
			sln = sln[1:]
			sln.sortSortLines()
			selected = sln[0]
			next = sln[1]
		}
	}
	datawriter.Flush()
}

/* Based on a glob pattern find the set of smallest 
 available files that meet the glob pattern and then
 merge them together opening up to maxConcurrentSeg
 together. */
func mergFiles(globPatt string,  maxConcurrentSeg int) {
}


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
 