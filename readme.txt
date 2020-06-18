oid mapper
----------------------------
Provides a high speed way of looking up the parent objects or views when a
fine grained child table s changed. he thought process is that we can leverage
the logic the top down walk based on foreign keys to walk the data hierarchy 
in a efficient way using our existing indexes.  With this ability we can use 
DB Audit functionality to identify every table record change and determine 
which views need to be re-published because they contain data from that table. 

Problem statement:

To implement a full data event bus we need a fast and efficient way 
to detect changes in published objects which any of their pieces of
data have changed.   When these objects are highly normalized in a RDBMS
it becomes a non trivial problem to detect which higher level objects
may be affected by a change at a lower level.   

We need an efficient way to map the changes at the smallest granular 
table back to the parent objects they are part of so we can schedule 
republishing a new version of the parent objects.  In RDBMS there is 
no direct link from child table data records to the parent objects 
especially when there are tie tables used to link unique child records 
such an address back to the parent objects that reference it. 

The goal is to be able to take 1..N child table name + object Key 
and provide a list of parent objects or parent view that should now
be considered dirty and need to be re-extracted or updated to reflect 
the change.  

This is a common problem when storing a JSON view object in Mongo 
or Elastic search when the master data remains in RDBMS.  

Since each child object is in a normalize form the audit or change log
on the child object do not directly contain the information for the
objects it is part of without a complex query.   In some cases we have 
found that deriving the parent object list is the slowest portion of 
ensuring timely refresh of extracted views for these objects. .

For example if I have an address object and person object and a company
object then a change to a single address may affect the JSON serialized
view of multiple people or companies located at the same address.  When 
the adress changes we need to find the all people and all the companies 
located at that address and re-publsih a changed view. 


Note: We still needs a way to identify new records in the composite child views 
but I think we can tell from the audit tables new records versus changed records 
which should make it possible to run a separate population process at that time. 

When running python on ubuntu then you must use python3 rather than python 
since that is what I tested the python scripts in. 


See:  reference.txt for hints on how to get postgress setup

Most commands are written to be ran from the command line using psql.
They assume that you are on linux or have cygwin bash installed.

####
# Machine Configs
####
*1 - Basic Tests ran on Dell laptop with SSD. Intel Core I5-8365U
     @ 1.6GHz, 1.9Ghtz,  160GB Ram,  Windows 10 enterprise
*2 - Indicates test ran on IBM P92, I5-3470@3.2GHzx4,
      16GB Ram, 256GB SSD. Ubuntu 18.04.4LTS
*3 - Super Micro OpenStack Ubuntu 16.04 LTS VM with 16 cores / 126GB mem
     RAM: DDR4-2933 2RX4 (16Gb) LP ECC RDIMM
     Intel CLX (Cascade Lake Gold) 6230 4/2P 20C/40T 2.1G 27.5M 10.4GT
     SSD SAS drives - WDC/HGST Ultrastar SS530 6.4TB SAS 12Gb
     With config changed recomended by pgtune.
*3L- Same Super Micro as *3 but with 899,993,602 records
*4 - Super Micro bare metal RHEL 7.8 with FIPS 140-2 encryption enabled
     RAM: DDR4-2666 2Rx4 (32Gb) ECC REG DIMM
     Intel SKL (SkyLake) 8160 8/4/2P 24C/48T 2.1G 33M 10.4GT
     NVMe drives - Samsung PM983, 3.84TB,NVMe,PCIe3.0x4
     With config changed recomended by pgtune.
*4L  Same as #4 but wil 1 billion records 
*4E  Same as #4L but shifted postgress to the encrypted volume

     


############
## Basic Setup
############
     
git clone https://github.com/joeatbayes/oid-mapper.git oidmap

Create the database, table, index for the mapping file
  psql -U test -f create_db.sql
  # This will remove and recreate the table


Generate sample oids mapping file:
  when in bash
  #python generateoids.py 10000000 data/stage/generated_oids.10m.map.txt 
  nohup time -o $PWD/time.generateoids.10m.txt python $PWD/generateoids.py 10000000 $PWD/data/stage/generated_oids.10m.map.txt
  # that has a number of synthetically
  # generated oids.  This file will consume about 
  # 2.7 Gig of storage space. You can reduce this by changing 
  # targetRows variable in generateoids.py  This generates some
  # combitorial records for many children composing a single master
  # the total number is random but on my test it actually produced 
  # This step can be skipped when real data of the same format is
  # available. 
  # *1 - 9m16.367s - 2.6G file created with 29,993,508 Recs
  #      29,993,508/((9*60)+6.367)= 54.9K recs pre sec.
  # *2 - 12m8.459s
  # *3 - 8m30.48s - 2.5G file created with 29,998,068 Recs 
  #      = 58.7K recs per sec.
  # *3L- 4h:52m:33sec 86GB with  
  #      1,029,008,744/((4*60*60)+(52*60)+33)= 58.62K recs per sec
  # *4 - 52m45s - db load 7.5G with 90K recsPerSec
  # *4L - 10h02m20s db load 86G with 1,028,993,082 recs.
  #       1,028,993,082/((10*60*60)+(2*60)+20)= 28.47K recs per sec
  # To generate roughly 1 billion records 
  nohup time -o $PWD/time.generateoids.340m.txt python $PWD/generateoids.py 343000000 $PWD/data/stage/generated_oids.340m.map.txt
  
Convert the oids file data file into a SQL command file to feed into postgress
  #time python create_db_load.py data/stage/generated_oids.10m.map.txt data/stage/db_load.10m.sql
  nohup time -o $PWD/time.create_db.10m.txt python create_db_load.py $PWD/data/stage/generated_oids.10m.map.txt $PWD/data/stage/db_load.10m.sql
  # replace test.map.txt with your input file. 
  # it will generate a file named in second parameter
  # this file will be slightly larger than the 
  # sum of the data passed in the list of input files.
  # *1 -2m54.24s = 148.8K per sec, 29,993,508/((2*60)+54.24)
  #     = 172.14K rec per sec.
  # *2 - 1m9.7s - 69 seconds = 43.3K per sec
  # *3 - 0m46.7s - 29,998,068 / ((0*60)+46.7) = 642.4K rec per sec.
  # *3L- 26:27.18 - 1,029,008,744/((0*60*60)+(26*60)+27) = 648.4K rec per sec
  # *4 - 2m47s - db load 7.5G  file with 90,007,790 recs = 179K per sec
  # *4L -33m35.29s  1,028,993,082/((0*60*60)+(33*60)+35) = 510.67K per sec
  #
  # To generate SQL to Load the aprox 1 billion records 
  nohup time -o $PWD/time.create_db.340m.txt python create_db_load.py data/stage/generated_oids.340m.map.txt data/stage/db_load.340m.sql
  
  
Load Postgress with the oids data 
  #time psql -U test -f data/stage/db_load.10m.sql
  nohup time -o $PWD/time.load_db.10m.txt psql -U test -f data/stage/db_load.10m.sql
  # On linux use time psql -f data/stage/db_load.sql 
  # to measure how long it takes. 
  # generates a file data/log/db_load.RESULT.txt
  # On windows you can use time command
  # by running a bash and then executing the 
  # same command you would on linux.  
  # On large files the would be to accomodate
  # ssh session timeout
  #     nohup time psql -f db_load.sql
  # *1 - 40m36.8s - 29,993,508/((40*60)+36.8) = 12.3K rec per sec
  #      with out of the box postgress configuration
  #      or  7.36K records per second.
  # *2 - 29.9 million records took 17m51.99s 
  #       or 27.8K records per second.
  #       With modified postgress config shown below.
  # *3  - 8m41.27sec - 29,998,068/((8*60)+41.3) = 57.5K rec per sec.
  #       after load dataase size is 10G.
  # *3L - 25h:08m:20s - 899,993,602 / ((25*60*60)+(8*60)+20)
  #       9,944 records per second.  Postgress Data usage 
  #       after inserts 189G.
  #  *4 - 28m16s - 90,007,790 / ((28*60) + 16) - 53,070 rec per sec
  #  *4L - 7h53m49s - 1,028,993,082/((7*60*60)+(53*60)+49) -36.2K rec per sec
  #        Postgress is 207G after load completes.
  #  *4E - 8h9m16s - 1,028,993,082/((8*60*60)+(9*60)+16) - 35.05K rec per sec
  # 
  # To load roughly 1B records into the datbase
  nohup time -o $PWD/time.load_db.340m.txt psql -U test -f data/stage/db_load.340m.sql
  

# Produce the test script for httpTester to exercise ther server
  #time python create_http_test_script.py data/stage/generated_oids.10m.map.txt data/stage/http-test-file.10m.txt
  nohup time -o $PWD/time.create_http_test.10m.txt python create_http_test_script.py $PWD/data/stage/generated_oids.10m.map.txt $PWD/data/stage/http-test-file.10m.txt
  # *1 - 1m35.64s - 29,993,508/((1*60)+34.64) = 316.9K rec per sec
  # *3 - 34.5s - 29,998,068 / ((0*60)+34.5) = 869.5K rec per sec.
  # *4 - 1m54.37s - 789.5K per sec

  # Produce test script for roughly 1B records
  nohup time -o $PWD/time.create_http_test.340m.txt python create_http_test_script.py $PWD/data/stage/generated_oids.340m.map.txt $PWD/data/stage/http-test-file.340m.txt
  # *3L - 34m50s - 1,028,993,082/((0*60*60)+(34*60)+50) = 492.34K recs per sec. 
  # *4L - 22m31.34s - 1,028,993,082/((0*60*60)+(22*60)+31.34) = 761.5K rec per sec.
  

## OBSOLETE ##
#Generate the file containing queries to test obtaining the distinct 
#parent oid, and table for every chiold oid in the system. 
#  #When in bash shell.
#  time -o $PWD/time.genSimple.10m.txt python generateSimpleQueries.py $PWD/data/stage/generated_oids.map.txt $PWD/data/stage/db_simple_queries.sql
#  # *1-On my laptop this took   266 seconds or about
#  #   112,406 per second.
#  #*2-
#  # *3 - 0m37.1s = 808K per sec
#  # *4 - 2m15s  = 666.7K per sec
  
## OBSOLETE ##  
#Run the query to select the parent oid and table for
#every child oid and table in the input data.
#  # When in bash
#  #time psql -f data/stage/db_simple_queries.sql
#  nohup time -o $PWD/time.db_simple.10m.txt psql -U test -f $PWD/data/stage/generated_oids.10m.map.txt
#  # This run will generate a file data/log/simple_query.RESULTS.txt
#  # that shows the results of every query.
#  # *1- ...m...s for 29.99 million records
#  #      or about ... milli-seconds per query.
#  # *2-   60m26s - (((60*60)+26)*1000)/29900000 = 0.12ms per oid
#  # *3-   60m26s - (((60*60)+26)*1000)/29900000 = 0.12ms per oid
#  # *4-
#  # A version to run in background for large files 
#  # nohup $(time psql -f $PWD/data/stage/generated_oids.lg.map.txt > $PWD/dbload.time.txt)
  
## OBSOLETE ##  
# Generate sql file to Query for the parents for every child OID in 
# the system using the IN clause.   
#  # when in bash
#  python generateInQueries.py data/stage/generated_oids.map.txt data/stage/db_in_queries.sql
#  # *1- 107 seconds to generate for 29.99 million rows. 
# # *2- 0m37.944s = 788.9K per sec
# # *3- 0m26.5s1128.3K per sec
# # *4- 1m39S = 909.2K per sec
  
## OBSOLETE ##  
# Run the query to select the parents for every child OID in the input
# file. 
#   # when in bash  
#   time psql -f data/stage/db_in_queries.sql 
#   # this generates a file in data/log/query.RESULTS.txt 
#   # *1-  20min23.26sec or 1220 seconds to run for 29.99 million 
#   #       child oids or about 0.0408 ms per child oid lookup.
#   # *2-  8m18s or (((8*60)+18)*1000)/29900000 = 0.0167ms per oid lookup
#   # *3-  5m7s or (((5*60)+7)*1000)/29900000 = 0.0103ms per oid lookup
#   # *4- 
 
   
   
#############
## FOR GOLANG TESTS
#############
# We use enviornment variables to obtain the user and password 
# needed to connect to postrgess when building the DB connect 
# string.   You will need to set these prior to running the test.
# Set  PGUSER = user you want to connect as
export PGUSER=SomeUserID
# Set PGPASS = password you want to connect to postgess with
export PGPASS=SomePassword
# See the Configure Ubuntu Linux below for instructions
# on getting most recent version of golang we used 
# some features in version 1.13

# Setup local enviornment
   #open shell windows to repodirectory/go
 
   # on windows
   set GOPATH=%cd%
   
   # on Linux
   export GOPATH=`pwd`
   export PATH=$GOPATH:$PATH


# Install the postress dependencies
    go get github.com/lib/pq


# Build the module to test a single query
   go build InQueryTest.go
   
# Build module to read input file and generate 
# queries n batches of a few using the IN clause
# and ieterate the results  

    go build inQueryFileTest.go
   
# run the module to test the DB access speed 
# via go.
  inQueryFileTest > t.t
  # you can look at times measured in GO using
  # head or tail. 
  # on my laptop this averaged between 1.9 and 2.3
  # milli-secondswhen using a IN query sized to 50 
  # client oids. This works out to about 0.04ms per
  # client oid. I tried settings from batch sizes
  # from 1 to 2500 and found a few that were slightly
  # faster but the larger batch size had more 
  # variablity.
  # While running single threaded the postgres server
  # consumes about
  # 17.3% of one 4 core I7.  In some latter records
  # it degraded from 20 to 27ms for a timing of about 
  # 0.05ms per client oid.
  # while the GO code was consuming between 5% and 7%
  
 
# Build the sample HTTP Server to test with Postgres back-end
  go build httpServer.go
  
# Run the sample HTTP server
  ./httpServer

# Test the sample HTTP Server.
      
   
# Download the http stress tester tool
go get -u -t "github.com/joeatbayes/http-stress-test/httpTest"



# Single threaded test against server
bin/httpTest -MaxThread=1 -in=../data/stage/http-test-file.10m.txt 
  # *1 - RecPerSec=145.3 - ((1/145.3)*1000)/25 = 0.275ms per oid
  # *2 - 19m33s - (((19*60)+30)*1000)/29900000 = 0.039ms per oid
  
  
 # *4L = ..m..s -  reported seminal rps = 14,576
          1,028,993,082/((0*60*60)+(..*60)+..) = ... recs per sec
          (((..*60)+..)*1000)/1,028,993,082= .....ms per oid
 
 # *4L = ..m..s -  reported start rp=4,745 end rps= ...
          1,028,993,082/((0*60*60)+(..*60)+..) = ... recs per sec
          (((..*60)+..)*1000)/1,028,993,082= .....ms per oid
 
 # *4E = 9h5m10s -  reported ending rps = 1209.9
          end elap = 0.58ms to 0.93ms mostly in 0.78ms.
          1,028,993,082/((9*60*60)+(5*60)+10) = 31.46K recs per sec
          (((545*60)+10)*1000)/1,028,993,082  = 0.0318ms per oid
 

# Slightly Heavier Load
bin/httpTest -MaxThread=2 -in=../data/stage/http-test-file.10m.txt 
  # *1 =  RecPerSec=625 - ((1/625)*1000)/25 = 0.0645ms per oid
  # *2 = 6m21s - (((6*60)+21)*1000)/29900000 = 0.01275ms per oid
 
  nohup time -o httpTest.340m.txt bin/httpTest -MaxThread=2 -in=../data/stage/http-test-file.340m.txt > t.t
 # *3L = ..m..s -  start reported rps=934.6  reported seminal rps = 
          start elap 1.22ms to 3.3ms
          1,028,993,082/((0*60*60)+(..*60)+..) = ... recs per sec
          (((..*60)+..)*1000)/1,028,993,082= .....ms per oid
 
 # *4L = ..m..s -  reported start rp=4,745 end rps= ...
          1,028,993,082/((0*60*60)+(..*60)+..) = ... recs per sec
          (((..*60)+..)*1000)/1,028,993,082= .....ms per oid
  
 # *4E = 4h32m21s -  reported seminal rps = 2,422
          elap = 0.76ms to 0.83ms
          1,028,993,082/((4*60*60)+(32*60)+21) = 92,9K recs per sec
          (((272*60)+21)*1000)/1,028,993,082 = 0.0159 ms per oid
 
 
  
# Medium Low load
bin/httpTest -MaxThread=4 -in=../data/stage/http-test-file.10m.txt 
  # *1 =  RecPerSec=1209 - ((1/1209)*1000)/25 = 0.033ms per oid
  # *2 = 7m15s - (((7*60)+15)*1000)/29900000= 0.01455ms per oid
  
  # nohup time -o httpTest.340m.txt bin/httpTest -MaxThread=4 -in=../data/stage/http-test-file.340m.txt > t.t
  # *3L = 230m40s -  reported seminal rps = 2859.6
          elap 1.079s to 2.265s
          1,028,993,082/((3*60*60)+(50*60)+40) = 74.35K. recs per sec
          (((230*60)+40)*1000)/1,028,993,082=  0.01345ms per oid
          
  # *4L = ..m..s -  reported seminal rps = 14,576
          1,028,993,082/((0*60*60)+(..*60)+..) = ... recs per sec
          (((..*60)+..)*1000)/1,028,993,082= .....ms per oid
          
  # *4E = 2h18m21s -  reported seminal rps = 4,768.6
          Elap 0.78ms to 1.17ms   
          1,028,993,082/((2*60*60)+(18*60)+21) = 123.96K recs per sec
          (((138*60)+18)*1000)/1,028,993,082= 0.00806ms per oid
 
# Medium load
  bin/httpTest -MaxThread=20 -in=../data/stage/http-test-file.10m.txt 
  # *1 =  RecPerSec=1833 - ((1/1833)*1000)/25 = 0.022ms per oid
          9m6.53 29,993,508/((9*60)+53) = 50.58K lookup per sec.
          (1/50579)*1000 = 0.0198ms per oid.
          last reported RPS=1952 = ((1/1952)*1000)/25 = 0.0205ms per oid.
  # *2 = 4m20s - (((4*60)+20)*1000)/29900000= 0.008ms per oid
  # *3 = 1m57 - (((1*60)+75)*1000)/29900000= 0.0045ms per oid

 
  # Same thing but reading the larger dataset inut.
  #bin/httpTest -MaxThread=20 -in=../data/stage/http-test-file.340m.txt 
  nohup time -o httpTest.340m.txt bin/httpTest -MaxThread=20 -in=../data/stage/http-test-file.340m.txt > t.t
  
  # *3L = 77m19s -  reported start rps = 14,576.  End rps=8,531
          1,028,993,082/(77*60)+19) = 222.7K recs per sec
          (((77*60)+19)*1000)/1,028,993,082= 0.0045ms per oid

  # *4L - Initial 16390 -  40m13s -  sentinal rps 16415 
          1,028,993,082/((0*60*60)+(40*60)+13) = 426.44K recs per sec
          (((40*60)+13)*1000)/1,028,993,082= 0.00235ms per oid
          
  # *4E - Initial 16390 -  48m29.83s -  sentinal rps 13608 
          1,028,993,082/((0*60*60)+(48*60)+29.83) = 353.63K recs per sec
          (((48*60)+29.83)*1000)/1,028,993,082= 0.00283ms per oid


# Stress Test Load
bin/httpTest -MaxThread=75 -in=../data/stage/http-test-file.10m.txt 
  # *1 = RecPerSec=1807 - ((1/1807)*1000)/25 = 0.022ms per oid
  # *2 = 3m37s - (((3*60)+37)*1000)/29900000= 0.0073ms per oid
         High variablity at this load with some requests 
         reaching 300ms response time when average is abount
         30ms. The top server load average is 48.3 with 
         tons of postgres processes consuming relatively
         small amounts.  
 # *3 = 1m31s - (((1*60)+31)*1000)/29900000= 0.003ms per oid
 # *4 = 3m54s - (((3*60)+54)*1000)/90,007,790 = 0.0026ms per oid
 
 nohup time -o httpTest.340m.txt bin/httpTest -MaxThread=75 -in=../data/stage/http-test-file.340m.txt > t.t
 
 # *3L = 71m57s -  reported start rps = 14,576 end rep=9,170
          1,028,993,082/((71*60)+57) = 238.36K recs per sec
          (((71*60)+57)*1000)/1,028,993,082=  0.0042ms per oid
 
 # *4L = 22m31.24s -  start RPS = 15,167.2 end rps = ...
          start elap= 1.03ms to 9.4ms
          load average: 34.52
          1,028,993,082/((0*60*60)+(22*60)+31.24) = 761.5K recs per sec
          (((22*60)+31)*1000)/1,028,993,082= 0.0013123ms per oid

 # *4E = 44m18s -  reported seminal rps = 14,898
          1,028,993,082/((0*60*60)+(44*60)+18) = 387.13K recs per sec
          ((44*60)+18)*1000)/1,028,993,082=  0.00258 ms per oid
 

# Stress Test Load abuse
bin/httpTest -MaxThread=250 -in=../data/stage/http-test-file.10m.txt 
  # *1 = RecPerSec=1770 - ((1/1770)*1000)/25 = 0.023ms per oid
  # *2 = 3m26s - (((3*60)+26)*1000)/29900000= 0.0069ms per oid
         High variablity at this load with some requests 
         reaching 3000ms response time with lot over 
         500ms.  There are still about 50% comming through
         in less than 25ms.  The top server load average is 40 with 
         tons of postgres processes consuming relatively
         small amounts.  
  # *3 = 1m13s - (((1*60)+13)*1000)/29900000= 0.00244ms per oid
   # Same load but with 1 billion records loaded. 
   nohup time -o httpTest.340m.txt bin/httpTest -MaxThread=250 -in=../data/stage/http-test-file.340m.txt > t.t
   
  # *3L = 67m54s -  reported end rps = 9713.6
          end elap = 15ms to 36ms
          1,028,993,082/(+(67*60)+54) = 252.57K recs per sec
          (((67*60)+54)*1000)/1,028,993,082= 0.00396ms per oid
 
  # *4L = 32m04.5 -  reported start rps=20,794 end rps = 20,591
          start elap = 2.14ms to 15.4ms  end elap= 13.7ms to 20.2ms
          load average = 48.96
          1,028,993,082/((0*60*60)+(32*60)+04.5) = 534,680K rec per sec
          (((33*60)+33.6)*1000)/1,028,993,082= 0.001957ms per oid
          
  # *4E = 33m52.4s -  reported seminal rps = 19,494 Rec per sec....
          1,028,993,082/((0*60*60)+(33*60)+52.4) = 506,294 oids recs per sec
          (((33*60)+52.4)*1000)/1,028,993,082= 0.001975ms per oid


bin/httpTest -MaxThread=400 -in=../data/stage/http-test-file.10m.txt 
  # *1 = RecPerSec=1690 - ((1/1690)*1000)/25 = 0.024ms per oid 
  # *3 = 1m7.6s - (((1*60)+7.6)*1000)/29900000= 0.00226ms per oid

  nohup time -o httpTest.340m.txt bin/httpTest -MaxThread=400 -in=../data/stage/http-test-file.340m.txt > t.t
 
  # *3L =  66m23 -  reported end rps =9,940
          end elap 35ms to 85ms
          1,028,993,082/((66*60)+23) = 258.35K recs per sec
          (((66*60)+23)*1000)/1,028,993,082= 0.00387 ms per oid
 
  # *4L = 24m50.68s -  start reported rps = 26,502, end rps= 26,592
          start elap 1.5ms to 32.7ms end elap = 15.5ms to 20.26ms
          load average = 61.6
          1,028,993,082/((0*60*60)+(24*60)+50.68) = 690,284.35 recs per sec
          (((24*60)+50.68)*1000)/1,028,993,082= 0.00145 ms per oid

  # *4E = 25m25.6 -  reported seminal rps = 25,982
          1,028,993,082/((0*60*60)+(25*60)+25.68) =  674,448.82 recs per sec
          (((25*60)+25.6)*1000)/1,028,993,082=  0.00148ms per oid
 

time -o time.httpTest600.txt bin/httpTest -MaxThread=600 -in=../data/stage/http-test-file.10m.txt > t.t
bin/httpTest -MaxThread=600 -in=../data/stage/http-test-file.10m.txt 
  # *1 = 9m6.97s 29,993,508/((9*60)+6.9) = 54.842K lookup per sec.
         (1/54842)*1000 = 0.018ms per oid.
         last reported RPS=2117.6 = ((1/2117.6)*1000)/25 = 0.0189ms per oid.
         which means the last records where slightly slower than first records
  # *3 = 1m5.2s - (((1*60)+5.2)*1000)/29900000= 0.00218ms per oid

  nohup time -o httpTest.340m.txt bin/httpTest -MaxThread=600 -in=../data/stage/http-test-file.340m.txt > t.t

 # *3L = 66m20s -  starting rps=9,304,  end rps= 9,947
         numfail=1172
         start elap= 13.9ms to 150.7ms end elap=60ms to 1503ms
         1,028,993,082/((66*60)+20) = 258.5K recs per sec
         (((66*60)+20)*1000)/1,028,993,082= 0.00387ms per oid
 
 # *4L = 22m.51.4s -  reported rps = 28,912
          end elap = 18.58ms to 41.6ms 
          1,028,993,082/((0*60*60)+(22*60)+51) = 750,542 recs per sec
          (((22*60)+51.4)*1000)/1,028,993,082= 0.00133 ms per oid
 
 
 # *4E = 21m44.2s -  reported seminal rps = 30,345
          1,028,993,082/((0*60*60)+(21*60)+44.2) = 788.98K recs per sec
          (((21*60)+44.2)*1000)/1,028,993,082=  0.001267ms per oid



# Note: When I tested with 750 connections I got errors in the server too many open 
#   files.   After further experimentation I found that it worked fine with 400 
#   MaxThread and fails at 450. I could bump this up by modifyng the 
#   linux limits but it is already CPU starved so would provide little
#   benefit.  The server does recover as soon as excess requests stop 
#   being made.  Worked fine up through 1,000 connections on rhel-7.



  
###############
### FOR JAVA TESTS
###############

  This section was tested with open JDK14
  downloaded from https://jdk.java.net/14/
  make sure the jdk/bin directory is added
  to your path enviornment variable.
  
  
  set class path to include 
    CLASSPATH=c:\PostgreSQL12\pgJDBC\postgresql-42.2.12.jar;
    # The JAR file was downloaded from
    #   https://jdbc.postgresql.org/download.html
    # See Java instructions for Ubuntu below 
    
  # Ensure the PGUSER and PGPASS enviornment
  # variables are set to reflect what has been
  # configured in postgres.
  
  Change directory to repo/go  on my machine this is 
  /jsoft/oidmap/java
  
  javac SimpleInQueryTest.java
  # produces the class file for this.
  
  java SimpleInQueryTest
  # Run the test should show some partbl found 
  
  javac InQueryFile.java
  # Build the test file that generates the SQL with
  # IN clause dynamically from input file.
  
  time java InQueryFile > t.t   
  #  Run the JAVA program to search on all child oids.
  #  this file reads the file "../test.map.txt"
  #  for input.  On my comuter for 29.99 million 
  #  records running single threaded it took
  #  *1 = 25m39.182S or 1539 seconds.   This works out 
  #       to 0.0513 seconds per client oid query. 
  #       When changed to 50 items in te in clause it
  #       dropped to 24m53 secons or 
  #  *2 = 14m0.9s - (((14*60)+0.94)*1000)/29900000= 0.0281ms per oid
  
  
###########
## Getting a Ubuntu install working
## Postgres 10 on Ubuntu 18.04 desktop
###########
sudo apt-get update
sudo apt-get -y upgrade
sudo apt-get install vim
sudo apt-get install git
git clone https://github.com/joeatbayes/oid-mapper.git oidmap
sudo chmod -R o+rwx /data/oidmap
sudo apt-get install openjdk-8-jdk
sudo apt-get install docker
sudo apt-get install postgresql postgresql-contrib

https://www.enterprisedb.com/download-postgresql-binaries
wget https://sbp.enterprisedb.com/getfile.jsp?fileid=12571&_ga=2.258807880.399051859.1591913766-484941476.1591913766

# If using machine with high speed storage on /data
sudo mv oidmap /data/oidmap
cd /data/oidmap

# Required package install
https://yallalabs.com/linux/how-to-install-and-use-postgresql-10-on-ubuntu-16-04/


# we need golang version 1.13 or newer for the strings.replaceAll
#   https://tecadmin.net/install-go-on-ubuntu/
wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz
#    NOTE:  On some machines using corporate proxies the wget
#    may not be allowed so you must obtian the file and transer 
#    using an alternative mechanism.  Or get the server added
#    to the approved list.
sudo tar -xvf go1.13.3.linux-amd64.tar.gz
sudo mv go /usr/local
rm go1.13.3.linux-amd64.tar.gz
# Required to allow GO to build for other architectures
sudo chmod -R 777 /usr/local/go/pkg

# Edit ~/.profile incude the following lines to let bash know where to look for GO
# cd ~
#  sudo vim .profile
export GOROOT=/usr/local/go
export GOPATH=$HOME/Projects/Proj1
export PATH=$GOPATH/bin:$GOROOT/bin:$PATH

# On my system this install postgres 10 If you want a 
#  newer version see https://pgdash.io/blog/postgres-11-getting-started.html

# Install the jdbc drivers for postgres
sudo apt-get install libpostgresql-jdbc-java
sudo apt-get update

# Configuring Postgress
# See: https://tecadmin.net/install-postgresql-server-on-ubuntu/
# Stop existing listener
 sudo systemctl stop postgresql  
# Check to be sure it actually stopped 
 sudo systemctl status postgresql

sudo mkdir /data2/pg12data
sudo chown jellsworth:games /data2/pg12data
/usr/lib/postgresql/12/bin/initdb -D /data2/pg12data
 sudo chmod -R a+rw /var/run/postgresql
/usr/lib/postgresql/12/bin/pg_ctl -D /data2/pg12data -l logfile start
  
# look to be sure database actually started 
  cat logfile 

/usr/lib/postgresql/12/bin/createdb jellsworth 

psql -c "CREATE ROLE test WITH LOGIN CREATEDB ENCRYPTED PASSWORD 'test';"
  # Change password to somethig you wuld actually want to use. 
psql -c "CREATE DATABASE test WITH OWNER test;"

# Edit the file 
   # /data2/pg12data/pg_hba.conf
   vi  /data2/pg12data/pg_hba.conf
   
   # Add the followiing line to allow test to login using md5 password   
   local   all             test                                    md5
   local   all             all                                     trust

# Now we must enable network listener.
# Edit file sudo vi /data2/pg12data/postgresql.conf
  vi /data2/pg12data/postgresql.conf
#  Change the line
#  Use the proper location for where your postgres is 
#  installed. 
   # listen_addresses = 'localhost'   
# to  the following by removing the leading #
   listen_addresses = 'localhost'   

/usr/lib/postgresql/12/bin/pg_ctl -D /data2/pg12data -l logfile restart
# look to be sure database actually started 
  cat logfile 

# Check to see if Posgres is actually listening on the portion
sudo netstat -plnt

# Create the oidmap database
psql -U test -f create_db.sql


#Edit pg_conf with these tunning settings
# Some will have to be uncommented by
# removing the #
# https://pgtune.leopard.in.ua/#/
# Linux 100GB Ram 14 cores on SSD
vi /data2/pg12data/postgresql.conf
  max_connections = 100
  shared_buffers = 25GB
  effective_cache_size = 75GB
  maintenance_work_mem = 2GB
  checkpoint_completion_target = 0.9
  wal_buffers = 16MB
  default_statistics_target = 100
  random_page_cost = 1.1
  effective_io_concurrency = 200
  work_mem = 32MB
  min_wal_size = 1GB
  max_wal_size = 4GB
  max_worker_processes = 14
  max_parallel_workers_per_gather = 4
  max_parallel_workers = 14
  max_parallel_maintenance_workers = 4

/usr/lib/postgresql/12/bin/pg_ctl -D /data2/pg12data -l logfile restart

# Execute the GoLang & Java Samples from
# Above
# Execute the Java Samples from Above

###  Execute section to Load and Test Data
###  in basic operation above. 
# Get the size of the oidmap database and related index
# Get size of database
/usr/lib/postgresql/12/bin/psql -c "SELECT pg_size_pretty(pg_database_size('oidmap'));"
# Get size of main table
/usr/lib/postgresql/12/bin/psql -U test -c "SELECT pg_size_pretty(pg_relation_size('omap'));" oidmap
# Get size of index
/usr/lib/postgresql/12/bin/psql -U test -c "SELECT pg_size_pretty(pg_indexes_size('ondx'));" oidmap
# Get # rows in main table
/usr/lib/postgresql/12/bin/psql -U test -c "SELECT COUNT(*) FROM omap;" oidmap


## Setup JDBC Driver for Postgres
cd oidmap/java
wget https://jdbc.postgresql.org/download/postgresql-42.2.13.jar
# Add oidmap/java/postgresql-42.2.13.jar to java class path
# by editing ~/.profile and adding the the line
sudo vi ~/.profile
# Add the line
export CLASSPATH=~/oidmap/java/postgresql-42.2.13.jar:$CLASSPATH
#activate new setting in current terminal
source ~/.profile
# ensure the environment variables PGUSER and PGPASS are set.
# to reflect what you have configured for postgres.
javac InQueryFile.java
java InQueryFile





####################
### Configure RHEL Box with Bare Metal Postgressql
####################
sudo yum check-update
sudo yum update
sudo yum install time.x86_64
sudo yum install vim
sudo yum install git
sudo yum install java-latest-openjdk.x86_64
sudo yum install docker-client-latest.x86_64
sudo yum install postgresql-jdbc.noarch 
  #  On my system this install postgres 9.5 If you want a 
  #  newer version see # https://linuxize.com/post/how-to-install-postgresql-on-centos-7/ 
  #  and https://pgdash.io/blog/postgres-11-getting-started.html

sudo yum install golang-vim.noarch golang.x86_64 
# NOTE: Golanguage 1.13 was already installed on my test box 

#####
## When the yum install can not work 
## packages can be installed directly
#####
# Remove old versions of postgress: 
#   sudo yum remove  postgresql.x86_64 postgresql-contrib.x86_64 postgre postgresql-server.x86_64 pg_top.x86_64 pg_view.noarch pgadmin3.x86_64
# From https://yum.postgresql.org/12/redhat/rhel-7-x86_64/repoview/
#  download version 12.3 rpm files for 
#   postgresql12-12.3-1PGDG.rhel7
#   postgresql12-libs-12.3-1PGDG.rhel7
#   postgresql12-server-12.3-1PGDG.rhel7
#   postgresql12-contrib-12.3-1PGDG.rhel7
#  And from https://yum.postgresql.org/12/redhat/rhel-7-x86_64/repoview/postgresql12.html
#   download the psql cleint 
#     postgresql12-12.3-1PGDG.rhel7.x86_64 
#   And transfer them to linux server 
 sudo rpm -i postgresql12-libs-12.3-1PGDG.rhel7.x86_64.rpm
 sudo rpm -i postgresql12-12.3-1PGDG.rhel7.x86_64.rpm
 sudo rpm -i postgresql12-server-12.3-1PGDG.rhel7.x86_64.rpm
 sudo rpm -i postgresql12-contrib-12.3-1PGDG.rhel7.x86_64.rpm
 

#Install Direcory is /usr/pgsql-12
sudo mkdir /data2/pg12data
sudo chown jellsworth:games /data2/pg12data
# initdb command explained https://www.postgresql.org/docs/12/app-initdb.html
/usr/pgsql-12/bin/initdb -D /data2/pg12data
sudo chmod a+rw /var/run/postgresql
/usr/pgsql-12/bin/pg_ctl -D /data2/pg12data -l logfile2 start
/usr/pgsql-12/bin/createdb jellsworth 
# createdb explained: https://www.postgresql.org/docs/12/app-createdb.html

psql -c "SELECT version();"
  # Should show the Database version which should be 12.3
psql -c "CREATE ROLE test WITH LOGIN CREATEDB ENCRYPTED PASSWORD 'test';"

psql -c "CREATE DATABASE test WITH OWNER test;"

# The Postgress config values are placed in the 
# named data directory used above.  In this instance
# They are /data2/pg12data/


# Edit the file /data2/pg12data/pg_hba.conf
   vi /data2/pg12data/pg_hba.conf
   # Add the following lines before the line that defines
   # local all so it looks like 
   local   all             test                                    md5
   local   all             all                                     trust


# Now we must enable network listener.
# Edit file vi /data2/pg12data/postgresql.conf
#  vi /data2/pg12data/postgresql.conf
#  Uncomment this line to allow java to connect
   # listen_addresses = 'localhost'   
# to  the following by removing the leading #
   listen_addresses = 'localhost'   
#  Uncomment the line 
# port = 5432

# Restart the server # https://www.postgresql.org/docs/12/app-pg-ctl.html
/usr/pgsql-12/bin/pg_ctl -D /data2/pg12data -l logfile2 restart

# Check to see if Postgres is actually listening on the portion
sudo netstat -plnt
  # You should see Postgres listening on port 5432
  
# Check Most recent logs in  /data2/pg12data/logto determine any 
# problem in startup.


# TEST that you can access the DB as local user
  psql -c "SELECT version();"
  # Should see a string showing the postgres version

# Test that you can access the DB as test user
  psql -U test -c "SELECT version();"
  # Should see a string showing the postgres version

# Use PGTune to generate new config settings for your machine
# modify postgresql.conf with the setting you generated.  Here is 
# what it generated for my machine. Update the file with the 
# following settings.  Some may be commented out initially
vi /data2/pg12data/postgresql.conf

# Update the file Joe reduced some of these settings 
# because it was pegging the system .
# during the database load.
max_connections = 100
shared_buffers = 100GB
effective_cache_size = 300GB
maintenance_work_mem = 2GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 128MB
min_wal_size = 1GB
max_wal_size = 4GB
max_worker_processes = 35
max_parallel_workers_per_gather = 4
max_parallel_workers = 35
max_parallel_maintenance_workers = 4

# Restart server after edits
/usr/pgsql-12/bin/pg_ctl -D /data2/pg12data -l logfile2 restart

# Create the oidmap database
psql -U test -f create_db.sql

# Assuming /data3 is a high speed storage 
# where you want to keep transient data 
# during the test
sudo mv oidmap /data3/oidmap
cd /data3/oidmap


# Execute the statements starting with 
# the create_db.sql under the section
# basic setup above

######
## Additonal Settings taht may be needed for some servers
######
# Modify Linux limits to allow more open files
  sudo vi /etc/security/limits.conf
  # add the following lines
  root soft     nproc          131072
  root hard     nproc          131072
  root soft     nofile         131072
  root hard     nofile         131072
  jellsworth    hard           maxlogins       50


# Fix the SSH session timeout issue
  sudo vi /etc/ssh_config
  # Add the lines 
  ClientAliveInterval 120
  ClientAliveCountMax 720
  
  
# Configure aide to ignore the data directories
# Modify the directories.
    update /etc/aide.conf 
      # and added 2 lines to the bottom -> 
        !/data Full 
        !/data2 Full 
     Run update-aide.conf


# Ask the VM Admin to allow access on port 9832

# Allow longer timeout on idle SSH sessions
##################
To $HOME/.bashrc add the following line
TMOUT=9000


##################
## Setup JDBC Driver for Postgres
##################
cd oidmap/java
wget https://jdbc.postgresql.org/download/postgresql-42.2.13.jar
# Add oidmap/java/postgresql-42.2.13.jar to java class path
# by editing ~/.profile and adding the the line
sudo vi ~/.profile
# Add the line
export CLASSPATH=~/oidmap/java/postgresql-42.2.13.jar:$CLASSPATH
#activate new setting in current terminal
source ~/.profile
# ensure the environment variables PGUSER and PGPASS are set.
# to reflect what you have configured for postgres.
javac InQueryFile.java
java InQueryFile




########################
### Modifications specifically for data from TST
########################
cd /data2/oidmap
python create_db_load.py ~/sub_enroll.data data/state/db_load.sql
  # Assumes the input data is at ~/sub_enroll.data
# *4 - 21.2 Sec

time psql -f data/stage/db_load.sql
# *4 - 7m41s 
 
time python generateInQueries.py data/stage/generated_oids.map.txt data/stage/db_in_queries.sql

# Do Not need this if testing http server access
time python generateSimpleQueries.py data/stage/generated_oids.map.txt data/stage/db_simple_queries.sql

# Do Not need this if testing http server access
time python create_http_test_script.py ~/sub_enroll.dat  ~/sub_enroll.http-test.txt
#  *4 - 14.68s

head $HOME/sub_enroll.http-test.txt

time go/bin/httpTest -MaxThread=1 -in=$HOME/sub_enroll.http-test.txt > t.t


