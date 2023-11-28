#! /bin/bash

printf "Starting gotor conainter"
docker run --network=host gotor
printf "\n"
printf "gotor container started on port :8081\n\n"