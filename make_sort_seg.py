""" Parse a DB load file and write a SQL
file to load that data into SQL. 
written compare to prepareInputFile.go 
to python speed for same data set. Where
Go is multi-process and python is limited
to a single core.
"""
import sys
import os

MaxBufLen = 15000000
MaxBufBytes = 1000000000

def quote(str):
    return "\'" + str + "\'"

def flush(buf, fiName):
  if len(buf) == 0:
    return
  buf.sort()
  with open(fiName, 'w') as fi:
    fi.writelines(buf)
    fi.close()

MaxBuffBytes = 999999999

# Process input file reading line by line.
# convert it into the proper field order
# sort and save to disk in segments 
def processFile(fname, foutName):
   fin = open(fname)
   hdr = fin.readline()
   buf = []
   segCnt = 0
   bytesBuff = 0
   while True:
     dline = fin.readline().strip()
     # Process single line
     if dline:
       flds = dline.split(",")
       #print("flds=", flds)
       partbl = flds[0]
       paroid = flds[1]
       chiltbl = flds[2]
       chiloid = flds[3]
       tstr =  chiloid + "," + paroid + "," + chiltbl + "," + partbl + "\n"
       bytesBuff += len(tstr)
       buf.append(tstr)
       if bytesBuff > MaxBuffBytes:
         flush(buf, foutName + str(segCnt).zfill(5) + ".pseg" )
         segCnt += 1
         bytesBuff = 0
         buf = []
     else: 
         break
   # Flush any left over at end of run
   flush(buff, foutName + str(segCnt) + ".pseg")

    
def printMsg(): 
  print("Usage:  python make_sortSeg.py inFiName outFiName")


# MAIN
if len(sys.argv) < 2:
    printMsg()
    raise ValueError('Please provide source file name')
    
fnameIn = sys.argv[1]
fnameOut = sys.argv[2]

print ("fnameIn=", fnameIn, " fnameOut=", fnameOut)
if not os.path.isfile(fnameIn):
   printMsg()
   raise ValueError("Could not find file " + str(fnameIn))
processFile(fnameIn, fnameOut)
                        


