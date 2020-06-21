# These installs where done with the visual package manager
#Install Visual Studio code

# Command line to install other needed packages
sudo swupd update
sudo swupd bundle-add wget vim
sudo swupd bundle-add clr-network-troubleshooter
sudo swupd bundle-add wget notepadqq
sudo swupd bundle-add postgresql12
sudo swupd bundle-add geany
sudo swupd bundle-add wget
sudo swupd bundle-add java13-basic
# NOTE:  Java command line is called java13 rather than java
#  and java compiler is javac13

# Enable local host lookup  # https://www.ionos.com/digitalguide/server/know-how/localhost/
sudo vi /etc/hosts
# Add the following lines 
127.0.0.1             localhost
::1                   localhost

#
# Download and install GO following instructions at: https://golang.org/doc/install?download=go1.14.4.linux-amd64.tar.gz
# Can not use default version becuase it is too old.
wget https://dl.google.com/go/go1.14.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.14.4.linux-amd64.tar.gz

vi $HOME/.profile
# Add the following lines
  export PATH=$PATH:/usr/local/go/bin
  export PATH=$PATH:/usr/libexec/postgresql12/
source  $HOME/.profile

sudo mkdir /data1/pg12data
sudo chown jwork:jwork /data1/pg12data
# initdb command explained https://www.postgresql.org/docs/12/app-initdb.html
/usr/libexec/postgresql12/initdb -D /data1/pg12data
sudo chmod a+rw /run/postgresql12/
/usr/libexec/postgresql12/pg_ctl -D /data1/pg12data -l logfile start
/usr/libexec/postgresql12/createdb jwork
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

# Use PGTune to generate new config settings for your machine
# modify postgresql.conf with the setting you generated.  Here is 
# what it generated for my machine. Update the file with the 
# following settings.  Some may be commented out initially
max_connections = 100
shared_buffers = 16GB
effective_cache_size = 48GB
maintenance_work_mem = 2GB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
work_mem = 20971kB
min_wal_size = 1GB
max_wal_size = 4GB
max_worker_processes = 8
max_parallel_workers_per_gather = 4
max_parallel_workers = 8
max_parallel_maintenance_workers = 4

# Restart the server # https://www.postgresql.org/docs/12/app-pg-ctl.html
pg_ctl -D /data1/pg12data -l logfile2 restart

psql -c "SELECT version();"

# If not already done cd to where oidmap 
# has been ran previously

# Create the oidmap database
psql -U test -f create_db.sql

# Follow instructions in main readme.txt
# under Basic Setup / Generate sample oids mapping file:


####################################################
## Build Postgress as Docker Image 
####################################################

# Install support for Local Docker
  #https://docs.01.org/clearlinux/latest/tutorials/docker.html
  #https://docs.01.org/clearlinux/latest/tutorials/kata.html
  sudo swupd bundle-add containers-virt
  sudo swupd bundle-add containers-basic

  sudo systemctl start docker
  sudo systemctl enable docker

  # To run container as Kata Container instead as normal container
  https://docs.01.org/clearlinux/latest/tutorials/kata.html#kata

  
  #######
  # With Default Ubuntu Docker Image
  #######
  # cd to directory where you downloaded oidmap  eg: $HOME/oidmap
  cd docker-ubuntu/
  docker build --tag docker-ubuntu . 
  
  # Most simple run.  Attached foreground and the ports
  # to access docker are not exposed. 
  sudo docker run --name bb docker-ubuntu:latest

  # Run with Port 5432 exposed as 5532 to allow 
  # external access.
  # cd to directory where you downloaded oidmap  eg: $HOME/oidmap
  cd docker-ubuntu/
  sudo docker rm --force bb
    # Delete last running image at name bb
  sudo docker run --publish 5532:5432 --name bb docker-ubuntu:latest
    # Publish traffic from host 5532 to containers port 5432
 
  # https://docs.docker.com/get-started/part2/
  # https://docs.01.org/clearlinux/latest/tutorials/kata.html#kata
  # https://docs.docker.com/engine/examples/postgresql_service/
    # container linking ports
	# connect from host
	# Using container volumes to nspec log files
	# Change config files of running image https://ligerlearn.com/how-to-edit-files-within-docker-containers/
	# Create docker config values using docker config https://medium.com/better-programming/about-using-docker-config-e967d4a74b83
	# Set enviornment variables using docker https://code.visualstudio.com/docs/remote/containers-advanced
	# Sample go based docker config for httpServer https://github.com/joeatbayes/metadata-forms-gui/blob/master/Dockerfile
	# Example of pulling from clearlinux as base for docker image https://github.com/clearlinux/dockerfiles/blob/master/mariadb/Dockerfile
	
	
	

  #######
  #  Build and Run Clear Linux Docker file
  #######
   # See: https://raw.githubusercontent.com/clearlinux/dockerfiles/master/postgres/Dockerfile which
   #  is the starting point for the Dockerfile we edited.
   cd $HOME/oidmap
   cd docker-postgress-clear/
   sudo docker build --tag docker-postgress-clear . 
   
   cd docker-postgress-clear/


  

