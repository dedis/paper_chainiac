# CHAINIAC

Chainiac is an update framework that provides decentralized enforcement of development and release processes, 
independent verification of source-to-binary correspondence, transparency via a collectively-signed update timeline, 
and efficient release validation by arbitrarily out-of-date clients.
This repository contains the bare-bone source code of the system and the code necessary to reproduce the experiments
in the corresponding paper.

## Installation

If you do not have Golang installed, start with

- Install [Golang](https://golang.org/doc/install)
- Set [`$GOPATH`](https://golang.org/doc/code.html#GOPATH) to point to your workspace directory
- Add `$GOPATH/bin` to `$PATH`


### Dependencies

To get Go dependencies of the project, execute

```
go get -u ./...
```

We use Python to extract metadata of Debian packages and snapshots of the Debian repository.
You need to install
1. Python 3.5 or later
2. BeautifulSoup4 and lxml libraries for processing HTML and XML files

## Simulation
Go to simulation/ and execute
```
go build .
```

## Reproducible builds
