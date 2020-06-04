""" Read a set of input files for the child oids
 and generate a SQL file that queries for the master
 records changed by those OID.   I am usng the therory
 that runing the commands directly from psql should yield
 the highest achivable performance since they should have
 optimized the command line client."""
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
       fout.write("SELECT DISTINCT paroid, partbl FROM omap WHERE omap.chiloid="
               + quote(chiloid) + " AND omap.chiltbl=" + quote(chiltbl) + ";\n")  
     else: 
         break
     

# MAIN
if len(sys.argv) < 2:
    raise ValueError('Please provide source file name')

fout = open("db_simple_queries.sql", "w")
fout.write("\\c oidmap\n\o simple_query.RESULTS.txt\n")
fnames = sys.argv[1:]
print ('fnames=', fnames)
for fname in fnames:
    if not os.path.isfile(fname):
        raise ValueError("Could not find file " + str(fname))
    processFile(fname, fout)
                        


