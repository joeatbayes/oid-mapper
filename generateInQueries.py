""" Read a set of input files for the child oids
 and generate a SQL file that queries for the master
 records changed by those OID.  This one uses an IN
 clause instead of the simple query to test relative
 performance of the two

 I am using the therory that runing the commands
 directly from psql should yield
 the highest achivable performance since they should have
 optimized the command line client.
"""

import sys
import os

def quote(str):
    return "\'" + str + "\'"

MaxInItems = 500
# Process input file reading line by line.
# Break it up into chunks and generate
# a psql file with separate insert statements
# for each chunk
def processFile(fname, fout):
   fin = open(fname)
   hdr = fin.readline()
   buf = []
   insStr = "INSERT INTO omap(chiloid, chiltbl, paroid, partbl) VALUES"
   while True:
     dline = fin.readline().strip()
     if dline:
       flds = dline.split(",")
       #print("flds=", flds)
       partbl = flds[0]
       paroid = flds[1]
       chiltbl = flds[2]
       chiloid = flds[3]
       buf.append(quote(chiloid))
       
       if (len(buf) > MaxInItems) or (not dline):
           if len(buf) > 0:
             fout.write("SELECT DISTINCT paroid, partbl FROM omap WHERE omap.chiloid IN ( ");                         
             sout = ", ".join(buf)
             fout.write(sout)
             fout.write(" );\n")
             
           buf = []                   
     else: 
         break
     
def printMsg(): 
  print("Usage:  python generateInQueries.py inFiName outFiName")


# MAIN
if len(sys.argv) < 3:
    raise ValueError('not enough parameters')

foutName = sys.argv[2]
fout = open(foutName, "w")
fout.write("\\c oidmap\n\o data/log/in_query.RESULTS.txt\n")
fnameIn = sys.argv[1]
print ("fnameIn=", fnameIn, "foutName=", foutName)
if not os.path.isfile(fnameIn):
    printMsg()
    raise ValueError("Could not find file " + str(fnameIn))
processFile(fnameIn, fout)
                        


