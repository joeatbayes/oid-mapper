oid mapper demonstration

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


See:  reference.txt for hints on how to get postgress setup

Most commands are written to be ran from the command line using psql.
They assume that you are on linux or have cygwin bash installed.

Create the database, table, index for the mapping file
  psql -f create_db.sh
  # This will remove and recreate the table


Generate sample oids mapping file:
  when in bash
  python generateoids.py
  # creates a file test.map.txt that has 10 million synthetically
  # generated oids.  This file will consume about 
  # 2.7 Gig of storage space. You can reduce this by changing 
  # targetRows variable in generateoids.py  This generates some
  # combitorial records for many children composing a single master
  # the total number is random but on my test it actually produced 
  # 29,995,113 million rows in 11.25 minutes.
  
Convert the oids file data file into a SQL command file to feed into postgress
  python create_db_load.py test.map.txt
  # replace test.map.txt with a list of your input files.
  # it will generate a file named db_load.out.sql
  # this file will be slightly larger than the 
  # sum of the data passed in the list of input files.
  # on my machine this took 201 seconds for 29.9 million 
  # records.
  
  
Load Postgress with the oids data 
  psql -f db_load.sql
  # On linux use time psql -f db_load.sql 
  # to measure how long it takes. 
  # generates a file db_load.RESULT.txt
  # On windows you can use time command
  # by running a bash and then executing the 
  # same command you would on linux.  On my machine
  # 100K records took 10.5 seconds.  For 29.9 million 
  # records it took  67 minutes 50 seconds. or 4060 seconds 
  # on my laptop with out of the box postgress configuration
  # This works out to 7364 records inserted per second.
  
  
Generate the file containing queries to test obtaining the distinct 
parent oid, and table for every chiold oid in the system. 
  #When in bash shell.
  time python generateSimpleQueries.py test.map.txt
  # On my laptop this took   266 seconds or about
  # 112,406 per second.
  # It generates the file db_simple_queries.sql
  
Run the query to select the parent oid and table for
every child oid and table in the input data.
  # When in bash
  time psql -f db_simple_queries.sql
  # This run will generate a file simple_query.RESULTS.txt
  # that shows the results of every query.
  # On my laptop this took ... seconds for 29.9 million records
  # or about ... milli-seconds per query.
  

Generate sql file to Query for the parents for every child OID in 
the system using the IN clause.   
  # when in bash
  python generateInQueries.py test.map.txt
  # generates a file db_in_queries.sql
  # From my laptop this took about 107 seconds to generate for 
  # 29.9 million rows. 
  


Run the query to select the parents for every child OID in the input
file. 
   # when in bash  
   time psql -f db_in_queries.sql
   # this generates a file in_query.RESULTS.txt on my laptop this
   # took  18m55.372s seconds to run for 29.9 million child oids or about
   #  0.038 ms per item or 26,427 Lookup per second.   
   
   
   
#############
## FOR GOLANG TESTS
#############

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
  