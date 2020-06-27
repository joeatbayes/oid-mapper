package main
/* Test Speed of reading a file using the buffer IO
  with smallest possible logic of scan and get the text
  compare to exReadAtLeast which does a read using 
  blocks of bytes
  
  time ./exReadBufio ../data/stage/generated_oids.340m.map.txt none
     Simple Read Measures 527 MiB/s on NUC from SATA SSD consuming 79% of one core 
	 complete in 2m46s. For 91319414195 bytes in 166 seconds.  =  
	 550116952 bytes per second = 550MB per second.   When system 
	 was idle dropped to 2m33s
	 
  write -  Read & Write strings 80% of CPU Read & Write String Peaks to 460MiB/s but
	   bounces around from 36 to 60 to 255 then  short times at peak 5m15s or 
	   2.17 times as long as simple read which makes sense if we were pushing
	   the device to do two things.  550MB per second / 2.17 = 253.5MB per second
	   When moved write to the NVME consumed 97% of core and time dropped to
	   3m5s.
	   
	writeba read & write but fetch bytes from reusable buffer 73% CPU
	shows 515Mib/s fairly stable. Reading from SATA SSD writing to 
	NVME ssd.  2m47 sec just about the same speed. Clearly the string
	conversion is having a material cost.  
	
	
	 time ./exReadBufio /home/jwork/index/340wrk-.merge.s-256.srt none
	   This is reading from the NVME 
	   reading 907MiB/s took  120% of 1 core.
	   Took 1m36.9s
	
	writeba when configured to read and write to NVME 
	   reading and writing at 560 MiB/s consuming 99% CPU
	   with chort drops to 60% CPU took 3m8.63s
	   
	writeba when configured to read NVME and write SATA SSD
	...reading and writing at 516 Mib/s while consuming 
	   83% of a CPU took 2m42s
	
	readmanybyte - reading from many segment files 1 line
	  2m10s Read 1 line at a time into a byte array.  
	  Read from Sata SSD Consuming 95% of 1 core with blips
	  back to 30%
	  
	rwmanybyte  - source on sata ssd, write to nvme disk write
	  600 MiB/s Read shows similar speed consuming 90% of 1 core
	  2m47s Not.  Copied 90,206,971,147 bytes
	  
	rwmanystr - 3m40s source on sata ssd, write to nvme 
	   445MiB/s with drops lower.   Consuming 119% of 1 core
	   
	rwmanysort -  - source on sata ssd, write to nvme
	   claims to be writing about 600MiB/s but fluctuating
	   a lot. CPU utilization is averagng about 82%
	   
*/

import (
	"fmt"
	"log"
	//"strings"
	"os"
	"bufio"
	"path/filepath"
	"bytes"
)

func procLine(line string) {
}

func readTest(fnameIn string) {
	r, err := os.Open(fnameIn)
	if err != nil {
		log.Fatal("error opening ", fnameIn, " err=", err)
	}
	scanner := bufio.NewScanner(r)
	
	for {
		more := scanner.Scan()
		str1 := scanner.Text()
		procLine(str1)
		if more == false { break }
	}
	fmt.Println("done")
}

/* Test most basic overhead of read bytes, write bytes */
func xx () {
}

/* Test most basic overhead of read a string
 write a string */
func readWriteStringsTest(fnameIn string, fnameOut string) {
	fout, foerr := os.OpenFile(fnameOut, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
			log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	r, err := os.Open(fnameIn)
	if err != nil {
		log.Fatal("error opening ", fnameIn, " err=", err)
	}
	scanner := bufio.NewScanner(r)
	for {
		more := scanner.Scan()
		str1 := scanner.Text()
		datawriter.WriteString(str1)
		if more == false { break }
	}
	fmt.Println("done")
}


/* Test most basic overhead of read a line
 at a time */
func readWriteByteArrTest(fnameIn string, fnameOut string) {
	fout, foerr := os.OpenFile(fnameOut, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
			log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	r, err := os.Open(fnameIn)
	if err != nil {
		log.Fatal("error opening ", fnameIn, " err=", err)
	}
	scanner := bufio.NewScanner(r)
	for {
		more := scanner.Scan()
		str1 := scanner.Bytes()
		datawriter.Write(str1)
		if more == false { break }
	}
	fmt.Println("done")
}

//  Test function that uses a glob path to 
//  load a list of files and simply reads one line
//  per file 
func readArrFilesByte(globPat string) {
	fnames, _ := filepath.Glob(globPat)
	if fnames == nil { 
		fmt.Println("L322 Found no files for globPat=", globPat)
	}
	numFile := len(fnames)
	fmt.Println("L132: initial fileList len=", numFile," files=", fnames)
	files := make([]*os.File, numFile)
	scanners := make([]*bufio.Scanner, numFile)
	//lines = make([]string, numFile)
	// Open our Array of Files 
	for ndx:= 0; ndx < numFile; ndx++ {
		fiPtr, err := os.Open(fnames[ndx])
		if err != nil {
			log.Fatalf("failed opening file %s: %s", fnames[ndx], err)
			return
		}
		defer fiPtr.Close()
		files[ndx] = fiPtr
		scanner := bufio.NewScanner(fiPtr)
		scanners[ndx] = scanner
	}
	fmt.Println("L144: All files open")
	allClosed := false
	bytesRead := 0
	linesRead := 0
	for allClosed != true {
		//fmt.Println("Outer Loop linesRead=", linesRead)
		allClosed = true
		for ndx:= 0; ndx < numFile; ndx++ {
			if scanners[ndx] == nil {
				// this file has reached EOF so skip
				fmt.Println("L155: Skip file closed")
				continue
			}
			allClosed = false
			more := scanners[ndx].Scan()
			str1 := scanners[ndx].Bytes()
			//fmt.Println("REad ndx=", ndx, " str1=", str1)
			blen := len(str1)
			bytesRead += blen
			linesRead += 1
			if more == false {
				files[ndx]=nil
				scanners[ndx]=nil
			}
	  }
	}
	fmt.Println("L167: bytesRead=", bytesRead, " linesRead=", linesRead)
}

//  Test function that uses a glob path to 
//  load a list of files and simply reads one line
//  per file 
func RWArrFilesStr(globPat string, fnameOut string) {
	fout, foerr := os.OpenFile(fnameOut, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
			log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	
	fnames, _ := filepath.Glob(globPat)
	if fnames == nil { 
		fmt.Println("L322 Found no files for globPat=", globPat)
	}
	numFile := len(fnames)
	fmt.Println("L132: initial fileList len=", numFile," files=", fnames)
	files := make([]*os.File, numFile)
	scanners := make([]*bufio.Scanner, numFile)
	//lines = make([]string, numFile)
	// Open our Array of Files 
	for ndx:= 0; ndx < numFile; ndx++ {
		fiPtr, err := os.Open(fnames[ndx])
		if err != nil {
			log.Fatalf("failed opening file %s: %s", fnames[ndx], err)
			return
		}
		defer fiPtr.Close()
		files[ndx] = fiPtr
		scanner := bufio.NewScanner(fiPtr)
		scanners[ndx] = scanner
	}
	fmt.Println("L144: All files open")
	allClosed := false
	bytesRead := 0
	linesRead := 0
	for allClosed != true {
		//fmt.Println("Outer Loop linesRead=", linesRead)
		allClosed = true
		for ndx:= 0; ndx < numFile; ndx++ {
			if scanners[ndx] == nil {
				// this file has reached EOF so skip
				fmt.Println("L155: Skip file closed")
				continue
			}
			allClosed = false
			more := scanners[ndx].Scan()
			str1 := scanners[ndx].Text()
			//fmt.Println("REad ndx=", ndx, " str1=", str1)
			datawriter.WriteString(str1)
			blen := len(str1)
			bytesRead += blen
			linesRead += 1
			if more == false {
				files[ndx]=nil
				scanners[ndx]=nil
			}
	  }
	}
	fmt.Println("L167: bytesRead=", bytesRead, " linesRead=", linesRead)
}



func RWArrFilesByte(globPat string, fnameOut string) {
	fout, foerr := os.OpenFile(fnameOut, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
			log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	
	fnames, _ := filepath.Glob(globPat)
	if fnames == nil { 
		fmt.Println("L322 Found no files for globPat=", globPat)
	}
	numFile := len(fnames)
	fmt.Println("L132: initial fileList len=", numFile," files=", fnames)
	files := make([]*os.File, numFile)
	scanners := make([]*bufio.Scanner, numFile)
	//lines = make([]string, numFile)
	// Open our Array of Files 
	for ndx:= 0; ndx < numFile; ndx++ {
		fiPtr, err := os.Open(fnames[ndx])
		if err != nil {
			log.Fatalf("failed opening file %s: %s", fnames[ndx], err)
			return
		}
		defer fiPtr.Close()
		files[ndx] = fiPtr
		scanner := bufio.NewScanner(fiPtr)
		scanners[ndx] = scanner
	}
	fmt.Println("L144: All files open")
	allClosed := false
	bytesRead := 0
	linesRead := 0
	for allClosed != true {
		//fmt.Println("Outer Loop linesRead=", linesRead)
		allClosed = true
		for ndx:= 0; ndx < numFile; ndx++ {
			if scanners[ndx] == nil {
				// this file has reached EOF so skip
				fmt.Println("L155: Skip file closed")
				continue
			}
			allClosed = false
			more := scanners[ndx].Scan()
			str1 := scanners[ndx].Bytes()
			//fmt.Println("REad ndx=", ndx, " str1=", str1)
			datawriter.Write(str1)
			blen := len(str1)
			bytesRead += blen
			linesRead += 1
			if more == false {
				files[ndx]=nil
				scanners[ndx]=nil
			}
	  }
	}
	fmt.Println("L167: bytesRead=", bytesRead, " linesRead=", linesRead)
}



func RWArrFilesSortStr(globPat string, fnameOut string) {
	fout, foerr := os.OpenFile(fnameOut, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
		log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	
	fnames, _ := filepath.Glob(globPat)
	if fnames == nil { 
		fmt.Println("L322 Found no files for globPat=", globPat)
	}
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
		defer fiPtr.Close()
		files[ndx] = fiPtr
		scanner := bufio.NewScanner(fiPtr)
		scanners[ndx] = scanner
	}
	fmt.Println("L144: All files open")
	bytesRead := 0
	linesRead := 0
	
	// Read a starter line from every file
	for ndx:= 0; ndx < numFile; ndx++ {
	 	if scanners[ndx] == nil {
			// this file has reached EOF so skip
			fmt.Println("L155: Skip file closed")
			//lines[ndx] = nil
			lines[ndx] = "~~"
			continue
		}
		more := scanners[ndx].Scan()
		//str1 := scanners[ndx].Bytes()
		str1 := scanners[ndx].Text()
		blen := len(str1)
		bytesRead += blen
		linesRead += 1
		//copy(lines[ndx],str1)
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
		for ndx:=0; ndx < numFile; ndx++ {
			//if lowest == nil {
			if lowest == "~~" {
				lowest = lines[ndx];
				lowestNdx = ndx;
			//} else if bytes.Compare(lowest, lines[ndx]) > 0 {
			} else if lowest > lines[ndx] {
				lowest = lines[ndx];
				lowestNdx = ndx;
			}
		}
		//if lowest == nil {
		if lowest == "~~" {
			// last line for all files must be nil which indicates
			// EOF has been reached for all files
			break;
		}
		
		// Based on the Lowest Identified Write that string 
		// to disk
		datawriter.WriteString(lowest)
		datawriter.WriteString("\n")
		if files[lowestNdx] == nil {
			// no more to read from this file
			lines[lowestNdx] = "~~"
			continue;
		} 
		
		if files[lowestNdx] == nil {
			lines[lowestNdx] = "~~"
		}
		more := scanners[lowestNdx].Scan()
		//str1 := scanners[lowestNdx].Bytes()
		str1 := scanners[lowestNdx].Text()
		if more == false {
			files[lowestNdx]=nil
			scanners[lowestNdx]=nil
		}
		//copy(lines[lowestNdx], str1) // copy from temp buffer which may get destroyed after next read
		lines[lowestNdx] = str1
		blen := len(str1)
		bytesRead += blen
		linesRead += 1
	}
	fmt.Println("L167: bytesRead=", bytesRead, " linesRead=", linesRead)
}


func RWArrFilesSortBA(globPat string, fnameOut string) {
	fout, foerr := os.OpenFile(fnameOut, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0644)
	if foerr != nil {
		log.Fatalf("failed creating file %s: %s", fnameOut, foerr)
	}
	datawriter := bufio.NewWriter(fout)
	defer fout.Close()
	
	fnames, _ := filepath.Glob(globPat)
	if fnames == nil { 
		fmt.Println("L322 Found no files for globPat=", globPat)
	}
	numFile := len(fnames)
	fmt.Println("L132: initial fileList len=", numFile," files=", fnames)
	files := make([]*os.File, numFile)
	scanners := make([]*bufio.Scanner, numFile)
	lines := make([][]byte, numFile)
	// initialize line buffers so we do not need to re-allocate
	for ndx:=0; ndx < numFile; ndx++ {
		lines[ndx] = make([]byte,6400)
	}
	//lines = make([]string, numFile)
	// Open our Array of Files 
	for ndx:= 0; ndx < numFile; ndx++ {
		fiPtr, err := os.Open(fnames[ndx])
		if err != nil {
			log.Fatalf("failed opening file %s: %s", fnames[ndx], err)
			return
		}
		defer fiPtr.Close()
		files[ndx] = fiPtr
		scanner := bufio.NewScanner(fiPtr)
		scanners[ndx] = scanner
	}
	fmt.Println("L144: All files open")
	bytesRead := 0
	linesRead := 0
	
	// Read a starter line from every file
	for ndx:= 0; ndx < numFile; ndx++ {
	 	if scanners[ndx] == nil {
			// this file has reached EOF so skip
			fmt.Println("L155: Skip file closed")
			lines[ndx] = nil
			continue
		}
		more := scanners[ndx].Scan()
		str1 := scanners[ndx].Bytes()
		blen := len(str1)
		bytesRead += blen
		linesRead += 1
		tline := lines[ndx]
		tline = tline[:0] // truncate current contents
		tline = append(tline, str1...)
		lines[ndx] = tline
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
		for ndx:=0; ndx < numFile; ndx++ {
			if lowest == nil {
				lowest = lines[ndx];
				lowestNdx = ndx;
			} else if bytes.Compare(lowest, lines[ndx]) > 0 {
				lowest = lines[ndx];
				lowestNdx = ndx;
			}
		}
		if lowest == nil {
			// last line for all files must be nil which indicates
			// EOF has been reached for all files
			break;
		}
		
		// Based on the Lowest Identified Write that string 
		// to disk
		datawriter.Write(lowest)
		datawriter.WriteString("\n")
		if files[lowestNdx] == nil {
			// no more to read from this file
			lines[lowestNdx] = nil
			continue;
		} 
		
		if files[lowestNdx] == nil {
			lines[lowestNdx] = nil
		}
		more := scanners[lowestNdx].Scan()
		str1 := scanners[lowestNdx].Bytes()
		if more == false {
			files[lowestNdx]=nil
			scanners[lowestNdx]=nil
		}
		
		copy(lines[lowestNdx], str1) 
		tline := lines[lowestNdx]
		tline = tline[:0] // clear buffer keep the storage
		tline = append(tline, str1...) // copy from temp buffer which may get destroyed after next read
		lines[lowestNdx] = tline
		
		blen := len(str1)
		bytesRead += blen
		linesRead += 1
	}
	fmt.Println("L167: bytesRead=", bytesRead, " linesRead=", linesRead)
}




func main() {
	fnameIn := os.Args[1]
	action  := os.Args[2] 
	foutName := "t.t99" // sata ssd
	//foutName := "/home/jwork/index/t.t99" // nvme
	
	fmt.Println("fnameIn=", fnameIn, " action=", action)
	
	if action == "write" {
		readWriteStringsTest(fnameIn, foutName) // nvme
	} else if action == "writeba"{
		readWriteByteArrTest(fnameIn, foutName)
	} else if action == "readmanybyte" {
		readArrFilesByte("../data/index/*.seg")
	} else if action == "rwmanybyte" {
		RWArrFilesByte("../data/index/*.seg", foutName)
	} else if action == "rwmanystr" {
		RWArrFilesStr("../data/index/*.seg", foutName)
	} else if action == "rwmanysort" {
		RWArrFilesSortStr("../data/index/*.seg", foutName)
	} else if action == "basic"{
		readTest(fnameIn)
	}
}
