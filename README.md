# goTor

[TorBot](https://github.com/DedSecInside/TorBoT) is an OSINT tool that allows you to scrape Tor sites and pull relevant information. It's currently ran using CLI which can be tedious for some users and can be offputting. To counter this issue, I'm designing a graphical representation of TorBot that behaves similarly but will be much easier to use. I'm rewriting TorBot in Golang instead of Python so I hope to see performance gains as well.

The only method that works currently is the `-l` argument from TorBot which lists all the associated links of a website. Before the repo will be near production ready, the rest of the arguments must be implemented as well.

## Getting Started

Currently goTor is only ran locally using webpack. To run goTor using webpack-dev-sever, simply run `npm run start`. This will launch goTor's front-end for you (and provide auto-reloads on active files if you're a developer). To start goTor's backend, you can simply run `go run server.go`, the server must be ran before you can actually use goTor fully.
