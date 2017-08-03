# CHAINIAC

Chainiac is an update framework that provides decentralized enforcement of development and release processes, 
independent verification of source-to-binary correspondence, transparency via a collectively-signed update timeline, 
and efficient release validation by arbitrarily out-of-date clients.
This repository contains the bare-bone source code of the system and the code necessary to reproduce the experiments
in the [corresponding paper](https://eprint.iacr.org/2017/648).

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


## Retrieving test data

First, run `./get_files.sh` to retrieve the list of the most downloaded Debian packages from `http://popcon.debian.org/`.
Then run
```
python3 get_repo_3.5.py
``` 
or 
```
python get_repo.py
```
to retrieve the snapshots of the Debian testing repository used in the experiments. This might take a while.


## Simulation
Go to simulation/ and execute
```
go build .
```

This will build a *simulation* binary. You can reproduce the experiments from the paper by running
the binary with a toml configuraiton file as an argument. 
The correspondance of the experiments and toml files is listed below:

- 
- 

## Reproducible builds
