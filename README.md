# gotor

### Status/Social Links
[![CircleCI](https://circleci.com/gh/KingAkeem/gotor.svg?style=svg)](https://circleci.com/gh/KingAkeem/gotor)
[![Open Source Helpers](https://www.codetriage.com/kingakeem/gotor/badges/users.svg)](https://www.codetriage.com/kingakeem/gotor)
[![](https://img.shields.io/badge/Made%20with-Golang-blue.svg?style=flat-square)]()


This is a HTTP REST API and command line program for webcrawling Tor (and non Tor) sites.

### Flags
#### Configuration of Tor client
- `-h` SOCKS5 proxy host, defaults to localhost
- `-p` SOCKS5 proxy port, defaults to 9050

#### REST
- `-server` Starts HTTP server that provides a REST API to the crawling mechanisms
- Current crawling mechanisms include: building relationship tree of links and getting the IP of the current tor client


#### CLI
- `-d` Searching for children nodes of links, defaults to 1
- `-o` Output destination, defaults to 'terminal' (recently added support for `excel` files)


### How it works
![Crawling drawio](https://user-images.githubusercontent.com/13573860/132710986-954b626d-5b42-4fc3-820a-737419690f35.png)
