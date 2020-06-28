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
	   
	rwmanysort -  - source on sata ssd, write to same 
	   claims to be writing about 40MiB/s but fluctuating
	   a lot. CPU utilization is averagng about 82%
	   uses string maniuplation took 33m2s/  
	   Drops to 31m20s when read from sata ssd write
	   to nvme.
	   
	rwmanysortbyte - 39m42s when reading from SATA SSD writing to same Sata SSD
	               - 39m6s when reading from SADA SSD writing to NVME SSD
	
	
	Thoughts if the basic search takes so long when reading and writing from
	disk.    from 3m30sec to 39m then it is roughly 10 X slower when
	scanning all 300 files and showing a IO froughly 46MiB then if we 
	where to reduce so we 1/10th the files to scan then what does it do 
	to IOPS.  If the answer is that we loose 90% of the difference from
	the scan then it may make sense to produce 10 files and then merge
	them in a seocnd pass.   When # of files was / 30 then speed 
	on disk jumped up to 278 MiBs and it took 11s.   When # files in
	compare loop was / 10 then speed jumped to about 180MiBs
	When dealing with 300 files dividing by 30 gives a compare
	compare set of 10 files.  That would reduce us to 30 output
	files.  When  / 10 it gives us a compare set of 30 files 
	and would produce 10 output files.   If we split the differnce
	and / 15 then it gives us compare set of 20 files and an
	output 20 files for the second phase.  The important part of 
	this is that we can run the the first phase parralell using
	all the cores and then run the last merge using only a single
	core.  When tested read sata SSD write to NVME with a 15 divider
	we averaged 191MiB/s and tool 30.254s to produce a file 5.9
	Gig long. 
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
		for ndx:=0; ndx < (numFile/15); ndx++ {			
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
	}
	fmt.Println("L167: bytesRead=", bytesRead, " linesRead=", linesRead)
}


func RWArrFilesSortBA(globPat string, fnameOut string) {
	LF := byte(12)
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
		datawriter.WriteByte(LF)
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
	//foutName := "t.t99" // sata ssd
	foutName := "/home/jwork/index/t.t99" // nvme
	
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
	} else if action == "rwmanysortbyte" {
		RWArrFilesSortBA("../data/index/*.seg", foutName)
	} else if action == "basic"{
		readTest(fnameIn)
	}
}
