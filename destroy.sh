# Stop and clean up previous environment
GOTOR_ID=$(docker ps | grep gotor | awk '{ print $1 }')
if [[ -n $GOTOR_ID ]]; then
    echo "Stopping gotor container"
	docker stop $GOTOR_ID
fi
TORPROXY_ID=$(docker ps | grep dperson/torproxy | awk '{ print $1 }')
if [[ -n $TORPROXY_ID ]]; then
	echo "Stopping dperson/torproxy container"
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
	echo "Removing docker network"
	docker network remove tor
fi
