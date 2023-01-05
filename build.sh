#!/bin/bash

source .env

# Check for a valid SOCKS5 configuration
if [[ -z $SOCKS5_HOST ]]; then
	echo "Error: You must specify a valid SOCKS5 hostname"
	exit 1
fi
if [[ -z $SOCKS5_PORT ]]; then
	echo "Error: You must specify a valid SOCKS5 port number"
	exit 1
fi

# Stop and clean up previous environment
GOTOR_ID=$(docker ps | grep gotor | awk '{ print $1 }')
if [[ -n $GOTOR_ID ]]; then
    echo "Stopping gotor container with ID $GOTOR_ID"
	docker stop $GOTOR_ID
fi
TORPROXY_ID=$(docker ps | grep dperson/torproxy | awk '{ print $1 }')
if [[ -n $TORPROXY_ID ]]; then
	echo "Stopping dperson/torproxy container with ID $TORPROXY_ID"
	docker stop $TORPROXY_ID
fi
TORPROXY_IMAGE=$(docker ps | grep dperson/torproxy)
if [[ -n $TORPROXY_IMAGE ]]; then
	echo "Removing dperson/torproxy image"
	docker rmi dperson/torproxy
fi
GOTOR_IMAGE=$(docker ps | grep gotor)
if [[ -n $GOTOR_IMAGE ]]; then
	echo "Removing gotor image"
	docker rmi  gotor
fi

TOR_NETWORK=$(docker network ls | grep tor)
if [[ -n $TOR_NETWORK ]]; then
	echo "Removing docker tor network"
	docker network remove tor
fi

# Restart tor if necessary
if [[ -n $USE_TOR && $USE_TOR = "true" ]]; then
	echo "Pulling and creating tor network"
	docker pull dperson/torproxy
	docker network create tor 

	echo "Starting dperson/torproxy container"
	docker run -d --rm -it --name tor_docker --network tor -p$SOCKS5_PORT:$SOCKS5_PORT dperson/torproxy 
fi

# Start main server
echo "Building and starting gotor container"
docker build -t gotor .
docker run -d --rm -it --network tor -p8081:8081 gotor
