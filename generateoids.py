""" Generate a set of oids that can be used to emulate client 
master lookup at various scale.  No effort made to avoid 
conflic when the oid generates the same oid. 
Tested with Python 3.8
"""

import uuid
import random
targetRows=300000000 # Minimum # rows to generate if targetOverlap always resolves to 1.
targetOverlap=5
master = ["person","person", "person", "company"]
# person repeated to cause it higher incidence in output file

children = ["address", "contact", "call", "complain"]
maxMaster = len(master) - 1
maxChild  = len(children) - 1

f = open("data/stage/generated_oids.map.txt", "w")
f.write("view_name,tview_oid,source_name,source_oid\n")

# Update the file with a bunch of master 
# reocords that have a randomized number
# of child records that coud update them. 
for rowndx in range(0, targetRows):
  numover = random.randint(1,targetOverlap)
  moid = str(uuid.uuid4())
  mtbl = master[random.randint(0,maxMaster)]
  for ovndx in range(0, numover):
    coid = str(uuid.uuid4())
    ctbl = children[random.randint(0,maxChild)]
    f.write(mtbl)
    f.write(",")
    f.write(moid)
    f.write(",")
    f.write(ctbl)
    f.write(",")
    f.write(coid)
    f.write("\n")

# Update the file with a bunch of master
# records where one child may update more
# than 1 master. 
f.close()
