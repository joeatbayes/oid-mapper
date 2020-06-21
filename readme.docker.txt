# Install & Enable Docker on Clear Linux Host
  #https://docs.01.org/clearlinux/latest/tutorials/docker.html
  #https://docs.01.org/clearlinux/latest/tutorials/kata.html
  sudo swupd bundle-add containers-virt
  sudo swupd bundle-add containers-basic

  sudo systemctl start docker
  sudo systemctl enable docker

  # To run container as Kata Container instead as normal container
  https://docs.01.org/clearlinux/latest/tutorials/kata.html#kata

  

####################################################
## Build Postgress as Docker Image 
####################################################


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
	
	

  #######
  #  Build and Run Clear Linux Docker file
  #######
   # See: https://raw.githubusercontent.com/clearlinux/dockerfiles/master/postgres/Dockerfile which
   #  is the starting point for the Dockerfile we edited.
   cd $HOME/oidmap
   cd docker-postgress-clear/
   sudo docker build --tag docker-postgress-clear . 
   
   cd docker-postgress-clear/


  
############
## Information that helped get Docker and Docker Files Working
############

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
	

