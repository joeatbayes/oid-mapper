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

When running python on ubuntu then you must use python3 rather than python 
since that is what I tested the python scripts in. 


See:  reference.txt for hints on how to get postgress setup

Most commands are written to be ran from the command line using psql.
They assume that you are on linux or have cygwin bash installed.

*1 - Basic Tests ran on Dell laptop with SSD.
*2 - Indicates test ran on IBM P92, I5-3470@3.2GHzx4, 
      16GB Ram, 256GB SSD. Ubuntu 18.04.4LTS
 
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
  #*2 - 12m8.459s
  
Convert the oids file data file into a SQL command file to feed into postgress
  python create_db_load.py test.map.txt
  # replace test.map.txt with a list of your input files.
  # it will generate a file named db_load.out.sql
  # this file will be slightly larger than the 
  # sum of the data passed in the list of input files.
  # *1-on my machine this took 201 seconds for 29.9 million 
  # records.
  #*2-1m9.7s - 69 seconds
  
  
Load Postgress with the oids data 
  psql -f db_load.sql
  # On linux use time psql -f db_load.sql 
  # to measure how long it takes. 
  # generates a file db_load.RESULT.txt
  # On windows you can use time command
  # by running a bash and then executing the 
  # same command you would on linux.  
  # *1 - 29.9 million  records it took  67m50s. or 4060 seconds 
  #      with out of the box postgress configuration
  #      or  7.36K records per second.
  # *2 - 29.9 million records took 17m51.99s 
  #       or 27.8K records per second.
  #       With modified postgress config shown below.
  
  
Generate the file containing queries to test obtaining the distinct 
parent oid, and table for every chiold oid in the system. 
  #When in bash shell.
  time python generateSimpleQueries.py test.map.txt
  # *1-On my laptop this took   266 seconds or about
  #   112,406 per second.
  #*2-
  # It generates the file db_simple_queries.sql
  
Run the query to select the parent oid and table for
every child oid and table in the input data.
  # When in bash
  time psql -f db_simple_queries.sql
  # This run will generate a file simple_query.RESULTS.txt
  # that shows the results of every query.
  # *1- ...m...s for 29.99 million records
  #      or about ... milli-seconds per query.
  # *2-   60m26s - (((60*60)+26)*1000)/29900000 = 0.12ms per oid
  

Generate sql file to Query for the parents for every child OID in 
the system using the IN clause.   
  # when in bash
  python generateInQueries.py test.map.txt
  # generates a file db_in_queries.sql
  # *1- 107 seconds to generate for 29.99 million rows. 
  # *2- 0m37.944s = 
  

Run the query to select the parents for every child OID in the input
file. 
   # when in bash  
   time psql -f db_in_queries.sql 
   # this generates a file in_query.RESULTS.txt 
   # *1-  20min23.26sec or 1220 seconds to run for 29.99 million 
   #       child oids or about 0.0408 ms per child oid lookup.
   # *2-  8m18s or (((8*60)+18)*1000)/29900000 = 0.0167ms per oid lookup
   #
   
   
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

# Produce the test script for httpTester to exercise ther server
python ../create_http_test_script.py ../test.map.txt http-test-file.txt

# Single threaded test against server
time bin/httpTest -MaxThread=1 -in=http-test-file.txt > t.t
  # *1 - 
  # *2 - 19m33s - (((19*60)+30)*1000)/29900000 = 0.039ms per oid

# Slightly Heavier Load
time bin/httpTest -MaxThread=2 -in=http-test-file.txt > t.t
  # *2 = 6m21s - (((6*60)+21)*1000)/29900000 = 0.01275ms per oid
  
# Medium Low load
time bin/httpTest -MaxThread=4 -in=http-test-file.txt > t.t
  # *2 = 7m15s - (((7*60)+15)*1000)/29900000= 0.01455ms per oid

# Medium load
time bin/httpTest -MaxThread=20 -in=http-test-file.txt > t.t
  # *2 = 8m30s - (((8*60)+30)*1000)/29900000= 0.0171ms per oid

# Stress Test Load
time bin/httpTest -MaxThread=75 -in=http-test-file.txt > t.t
  # *2 = 11m15s - (((11*60)+15)*1000)/29900000= 0.0226ms per oid
         High variablity at this load with some requests 
         reaching 300ms response time when average is abount
         30ms. The top server load average is 48.3 with 
         tons of postgres processes consuming relatively
         small amounts.  

# Stress Test Load abuse
time bin/httpTest -MaxThread=250 -in=http-test-file.txt > t.t
  # *2 = 10m22s - (((10*60)+22)*1000)/29900000= 0.0208ms per oid
         High variablity at this load with some requests 
         reaching 3000ms response time with lot over 
         500ms.  There are still about 50% comming through
         in less than 25ms.  The top server load average is 40 with 
         tons of postgres processes consuming relatively
         small amounts.  

# Note: When I tested with 750 connections I got errors in the server too many open #   files.   After further experimentation I found that it worked fine with 400 
#   MaxThread and fails at 450. I could bump this up by modifyng the 
#   linux limits but it is already CPU starved so would provide little
#   benefit.  The server does recover as soon as excess requests stop 
#   being made.

  
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
  #  25m39.182S or 1539 seconds.   This works out 
  #  to 0.0513 seconds per client oid query. 
  #  When changed to 50 items in te in clause it
  #  dropped to 24m53 secons or 
  
  
###########
## Getting a Ubuntu install working
## Postgres 10 on Ubuntu 18.04 desktop
###########
sudo apt-get install vim
sudo apt-get install git
git clone https://github.com/joeatbayes/oid-mapper.git

# If using machine with high speed storage on /data
sudo mv oidmap /data/oidmap
cd /data/oidmap

# Required package install
sudo apt-get update
sudo apt-get -y upgrade

# we need golang version 1.13 or newer for the strings.replaceAll
#   https://tecadmin.net/install-go-on-ubuntu/
wget https://dl.google.com/go/go1.13.3.linux-amd64.tar.gz
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


sudo apt-get install openjdk-8-jdk
sudo apt-get install docker
sudo apt-get install postgresql postgresql-contrib
# On my system this install postgres 10 If you want a 
#  newer version see https://pgdash.io/blog/postgres-11-getting-started.html

# Install the jdbc drivers for postgres
sudo apt-get install libpostgresql-jdbc-java

sudo apt-get update

# Configuring Postgress
# See: https://tecadmin.net/install-postgresql-server-on-ubuntu/
sudo su - postgres
psql
# with postgres-# prompt
   CREATE ROLE test WITH LOGIN CREATEDB ENCRYPTED PASSWORD 'test'
   # To show the users in the system to verify your user was created
   \du
   CREATE DATABASE jwork WITH OWNER jwork;
   CREATE DATABASE test WITH OWNER test;
   \q
#  The Create database is done for each user created since the 
#  default database the system will attempt to connect to is same
#  name as user.     Where I created a user jwork you would create
#  one with the name of the linux user you want to use to access
#  postgres.

# return to your linux user.
exit

# On my ubuntu desktop  Version 18.04.4 LTS
# I found the ubuntu settings at /etc/postgresql/10/main
# You may have to look elsewhere.   If you desparate 
# to find yours tyr the following
#   sudo find / | grep posgresssql.conf

# Edit the file 
   /etc/postgresql/10/main/pg_hba.conf
   # sudo vi  /etc/postgresql/10/main/pg_hba.conf
   # Add the followiing lines after the line that says local all postgres peer
   # so the file looks like
   local   all             postgres                                peer
   local   all             jwork                                   md5
   local   all             test                                    md5

# Now we must enable network listener.
# Edit file sudo vi /etc/postgresql/10/main/postgresql.conf
#  sudo vi /etc/postgresql/10/main/postgresql.conf
#  Change the line
   # listen_addresses = 'localhost'   
# to  the following by removing the leading #
   listen_addresses = 'localhost'   


# Restart the server 
 sudo service postgresql restart

# Check to see if postgres server is running 
service --status-all
# Check to see if Posgres is actually listening on the portion
sudo netstat -plnt


# If server does not seem to be responding then
# check the log file.  On my system I used the 
# command 
tail -n500 /var/log/postgresql/*
# If you see a line containing " LOG:  database system is ready to accept connections" then you should be able to now log in using psql

# Create the oidmap database
psql -f create_db.sql

# Use PGTune to generate new config settings for your machine
# modify postgresql.conf with the setting you generated.  Here is 
# what it generated for my machine
sudo vi /etc/postgresql/10/main/postgresql.conf
# https://pgtune.leopard.in.ua/#/
# DB Version: 10
# OS Type: linux
# DB Type: oltp
# Total Memory (RAM): 16 GB
# CPUs num: 4
# Data Storage: ssd

# Joe reduced some of these settings 
# because it was pegging the system .
# during the database load.
max_connections = 300
shared_buffers = 3GB
effective_cache_size = 10GB
maintenance_work_mem = 1GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 6990kB
min_wal_size = 2GB
max_wal_size = 6GB
max_worker_processes = 3
max_parallel_workers_per_gather = 2
max_parallel_workers = 3
# END OF postgresql.conf edits.
sudo service postgresql restart

# These Generation script next 5 lines 
# are the same as those above. included 
# here to  keep continutiy in this section
time python generateoids.py
time python create_db_load.py test.map.txt
time psql -f db_simple_queries.sql
time python generateInQueries.py test.map.txt
 
time psql -f db_load.sql
# This utility takes some time You can check progress
#  with the following
    # Counts the lines which is roughly equal to the number
	# of 50,000 record inserts completed
    wc -l db_load.RESULT.txt
	# Shows last lines in the DB
	tail db_load.RESULT.txt
	
time psql -f db_in_queries.sql
time psql -f db_in_queries.sql
 
 # Get the size of the oidmap database and related index
 psql
 \c oidmap
// Get size of database
SELECT pg_size_pretty(pg_database_size('oidmap'));
// Get size of main table
SELECT pg_size_pretty(pg_relation_size('omap'));
// Get size of index
SELECT pg_size_pretty(pg_indexes_size('index_empid'));
// Get # rows in main table
SELECT COUNT(*) FROM omap;



# TODO:   GoLang and Java Samples need to pull
#   User from the local user name to allow proper execution
#
# Execute the GoLang & Java Samples from
# Above

# Execute the Java Samples from Above

