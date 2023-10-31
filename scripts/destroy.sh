# Stop and clean up previous environment
GOTOR_ID=$(docker ps | grep gotor | awk '{ print $1 }')
if [[ -n $GOTOR_ID ]]; then
    echo "Stopping gotor container"
	docker stop $GOTOR_ID

	echo "Removing gotor image"
	docker image rm gotor
fi
TORPROXY_ID=$(docker ps | grep dperson/torproxy | awk '{ print $1 }')
if [[ -n $TORPROXY_ID ]]; then
	echo "Stopping dperson/torproxy container"
	docker stop $TORPROXY_ID

	echo "Removing dperson/torproxy image"
	docker image rm dperson/torproxy
fi

TOR_NETWORK=$(docker network ls | grep tor)
if [[ -n $TOR_NETWORK ]]; then
	echo "Removing Tor docker network"
	docker network remove tor
fi
