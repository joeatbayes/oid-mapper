package main
 
import (
	"bufio"
	//"fmt"
	"log"
	"os"
	//"strings"
	"sort"
	//"strconv"
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
}

type SortLines []SortLine


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
func mergFilesPat(globPatt string,  maxConcurrentSeg int) {
	// Get a list of files
	
	// Find the set of smallest files smaller than maxConcurrentSeg
	
	// Merge the files into a larger file 
	// remove the segment files. 
	
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


func main() {
	
}
 