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
echo "Removing docker tor network"
docker network remove tor
