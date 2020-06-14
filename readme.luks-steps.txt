Steps Modified for LUKS volume

# Stop postgress if already running in the other directory.
/usr/pgsql-12/bin/pg_ctl -D /data2/pg12data -l logfile2 stop

# Assumes you already have a working
# Postgres 12 from the main readme working.
# and Install Direcory is /usr/pgsql-12

sudo mkdir /data1/pg12data
sudo chown jellsworth:games /data1/pg12data
# initdb command explained https://www.postgresql.org/docs/12/app-initdb.html
/usr/pgsql-12/bin/initdb -D /data1/pg12data
sudo chmod a+rw /var/run/postgresql
/usr/pgsql-12/bin/pg_ctl -D /data1/pg12data -l logfile2 start
/usr/pgsql-12/bin/createdb jellsworth 
# createdb explained: https://www.postgresql.org/docs/12/app-createdb.html

psql -c "SELECT version();"
  # Should show the Database version which should be 12.3
psql -c "CREATE ROLE test WITH LOGIN CREATEDB ENCRYPTED PASSWORD 'test';"

psql -c "CREATE DATABASE test WITH OWNER test;"

# The Postgress config values are placed in the 
# named data directory used above.  In this instance
# They are /data1/pg12data/


# Edit the file /data2/pg12data/pg_hba.conf
   vi /data1/pg12data/pg_hba.conf
   # Add the following lines before the line that defines
   # local all so it looks like 
   local   all             test                                    md5
   local   all             all                                     trust


# Now we must enable network listener.
# Edit file vi /data1/pg12data/postgresql.conf
#  vi /data1/pg12data/postgresql.conf
#  Uncomment this line to allow java to connect
   # listen_addresses = 'localhost'   
# to  the following by removing the leading #
   listen_addresses = 'localhost'   
#  Uncomment the line 
# port = 5432

# Restart the server # https://www.postgresql.org/docs/12/app-pg-ctl.html
/usr/pgsql-12/bin/pg_ctl -D /data1/pg12data -l logfile2 restart

# Check to see if Postgres is actually listening on the portion
sudo netstat -plnt
  # You should see Postgres listening on port 5432
  
# Check Most recent logs in  /data1/pg12data/logto determine any 
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
vi /data1/pg12data/postgresql.conf

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
/usr/pgsql-12/bin/pg_ctl -D /data1/pg12data -l logfile2 restart

# If not already done cd to where oidmap 
# has been ran previously

# Create the oidmap database
psql -U test -f create_db.sql


# Execute the statements starting with 
# the create_db.sql under the section
# basic setup above.  You can skip the
# portions that generated the oids and
# other text files if already created for
# other tests.  The first step in that instance
# will be excuting the psql statement to 
# load the DB.
# nohup time -o $PWD/time.load_db.340m.txt psql -U test -f data/stage/db_load.340m.sql
