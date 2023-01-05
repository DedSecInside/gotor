#!/bin/bash

source .env

if [[ -z $SOCKS5_PORT ]]; then
	echo "Error: You must specify a valid SOCKS5 port number"
	exit 1
fi

# Stop and clean up previous environment
./destroy.sh

# Restart tor if necessary
if [[ -n $USE_TOR && $USE_TOR = "true" ]]; then
	echo "Pulling and creating tor network"
	docker pull dperson/torproxy
	docker network create tor 

	echo "Starting dperson/torproxy container"
	docker run -d --rm -it --name tor_service --network tor -p$SOCKS5_PORT:$SOCKS5_PORT dperson/torproxy 
fi

# Start main server
echo "Building and starting gotor container"
docker build -t gotor .
docker run -d --rm -it --network tor -p8081:8081 gotor
