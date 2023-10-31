#!/bin/bash

printf "Building gotor image"
docker build -t gotor .
printf "\n"
printf "gotor image has been built.\n\n"
