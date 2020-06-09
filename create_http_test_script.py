""" Parse a DB load file and write a SQL
file to load that data into SQL. """
import sys
import os
import json

MaxBufLen = 25
lineCnt = 0

def quote(str):
    return "\'" + str + "\'"

# Process input file reading line by line.
# Break it up into chunks and generate
# a psql file with separate insert statements
# for each chunk
def processFile(fname, fout):
   global lineCnt
   fin = open(fname)
   hdr = fin.readline()
   buf = []
   while True:
     dline = fin.readline().strip()
     lineCnt += 1
     if dline:
       flds = dline.split(",")
       #print("flds=", flds)
       partbl = flds[0]
       paroid = flds[1]
       chiltbl = flds[2]
       chiloid = flds[3]
       buf.append(chiloid)
       if (len(buf) > MaxBufLen) or (not dline):
           if len(buf) > 0:
             turi = "http://127.0.0.1:9832/oidmap?keys=" + ",".join(buf)
             jobj =  {  "id" : str(lineCnt),   
                         "verb" : "GET", 
                         "uri" : turi, 
                         "expected" : 200,
                         "rematch" : "--Num Match"}
             jstr = json.dumps(jobj);
             fout.write(jstr)
             fout.write("\n#END\n")
           buf = []                   
     else: 
         break
     

# MAIN
if len(sys.argv) < 2:
    raise ValueError('Please provide source file name')
fnameIn = sys.argv[1]
fnameOut = sys.argv[2]
fout = open(fnameOut, "w")
if not os.path.isfile(fnameIn):
        raise ValueError("Could not find file " + str(fnameIn))
processFile(fnameIn, fout)
                        


