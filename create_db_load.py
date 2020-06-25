""" Parse a DB load file and write a SQL
file to load that data into SQL. """
import sys
import os

MaxBufLen = 15000000
MaxBufBytes = 1000000000

def quote(str):
    return "\'" + str + "\'"

# Process input file reading line by line.
# Break it up into chunks and generate
# a psql file with separate insert statements
# for each chunk
def processFile(fname, fout):
   fin = open(fname)
   hdr = fin.readline()
   buf = []
   while True:
     dline = fin.readline().strip()
     # Flush if EOF or too many items
     if (len(buf) > MaxBufLen) or (not dline):
           if len(buf) > 0:
             fout.write(insStr + "\n");             
             sout = ",\n".join(buf)
             fout.write(sout)
             fout.write(";\n\n")             
           buf = []      
           
     # Process single line
     if dline:
       flds = dline.split(",")
       #print("flds=", flds)
       partbl = flds[0]
       paroid = flds[1]
       chiltbl = flds[2]
       chiloid = flds[3]
       buf.append( chiloid + "," + paroid + "," + chiltbl + "," + partbl )
                    
     else: 
         break

def printMsg(): 
  print("Usage:  python create_db_load.py inFiName outFiName")


# MAIN
if len(sys.argv) < 3:
    printMsg()
    raise ValueError('Please provide source file name')
    
fnameIn = sys.argv[1]
fnameOut = sys.argv[2]
fout = open(fnameOut, "w")
fout.write("\\c oidmap\n\o db_load.RESULT.txt\n")

print ("fnameIn=", fnameIn, " fnameOut=", fnameOut)
if not os.path.isfile(fnameIn):
   printMsg()
   raise ValueError("Could not find file " + str(fnameIn))
processFile(fnameIn, fout)
                        


