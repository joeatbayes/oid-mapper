""" Parse a DB load file and write a SQL
file to load that data into SQL. """
import sys
import os

MaxBufLen = 25

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
     if dline:
       flds = dline.split(",")
       #print("flds=", flds)
       partbl = flds[0]
       paroid = flds[1]
       chiltbl = flds[2]
       chiloid = flds[3]
       buf.append( "(" + quote(chiloid) + "," + quote(chiltbl) + "," + quote(paroid) + "," + quote(partbl) + ")")
       if (len(buf) > MaxBufLen) or (not dline):
           if len(buf) > 0:
             fout.write(insStr + "\n");             
             sout = ",\n".join(buf)
             fout.write(sout)
             fout.write(";\n\n")
             
           buf = []                   
     else: 
         break
     

# MAIN
if len(sys.argv) < 2:
    raise ValueError('Please provide source file name')

fout = open("db_load.sql", "w")
fout.write("\\c oidmap\n\o db_load.RESULT.txt\n")
fnameIn = sys.argv[1]
fnameOut = sys.argv[2]
print ('fnames=', fnames)
for fname in fnames:
    if not os.path.isfile(fname):
        raise ValueError("Could not find file " + str(fname))
    processFile(fname, fout)
                        


