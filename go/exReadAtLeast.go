package main
/* Test Speed of reading a file 64K chunks at a time.
  want this to compare to speed of reading with 
  buffered scanner
  
  time ./exReadAtLeast ../data/stage/generated_oids.340m.map.txt none
     Simple REad Measures 527.7 MiB/s on NUC from SATA SSD consuming 19% of one core
     Read Parse Line us 70% of CPU do do simple scan for LF but still reading at 519MiB/s
  */

import (
	"fmt"
	"io"
	"log"
	//"strings"
	"os"
)

var bufSize int = 32768
var lineBuf []byte = make([]byte, bufSize+1)

func procLine(start int, stop int, buf []byte) {
}

/* Scan a input buffer seeking ASCII LF whenever
 it occurs sent the bytes between the last call
 or start of string and current offset to procLine 
 written to test CPU impact of most simple scan
 to parse lines out 1 at a time. 
*/
func parseLines(numBytes int, buf []byte) {
	lineOff := 0
	off := 0
	for off = 0; off < numBytes; off++ {
		if buf[off] == 12 {
			procLine(lineOff, off -1, buf)
			lineOff = 0
		}
		lineBuf[lineOff] = buf[off]
		lineOff++
	}
	if lineOff < off {
	  procLine(lineOff, off-1, buf)
	}
	// TODO: Move left over buffer items if did not end
	// with lf to front of buffer 
}

func main() {
	fnameIn := os.Args[1]
	action  := os.Args[2] 
	r, err := os.Open(fnameIn)
	if err != nil {
		log.Fatal("error opening ", fnameIn, " err=", err)
	}
	
	
	buf := make([]byte, bufSize+1)

	
	for {
		numRead, err := io.ReadAtLeast(r, buf, bufSize)
		if err == io.EOF {
			fmt.Println("Reached EOF")
			log.Fatal(err)
			break
		} else if err != nil {
			log.Fatal("Err reading file err=", err)
		}
		if numRead < bufSize {
			fmt.Println("Received less than full buffer received=", numRead, " asked=", bufSize)
			break
		}
		if action == "lines" {
			parseLines(numRead, buf)
		}
	}
	fmt.Println("done")
}
