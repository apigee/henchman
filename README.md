

# henchman [![Circle CI](https://circleci.com/gh/apigee/henchman/tree/master.svg?style=svg)](https://circleci.com/gh/apigee/henchman/tree/master)

## What is Henchman
Henchman is an orchestration and automation tool written in Golang, inspired by Ansible with support for custom transports and inventories.

Check out the [wiki](https://github.com/apigee/henchman/wiki) to learn more.

## How Henchman Works
Henchman executes a plan (a collection of tasks) on a given set of machines.  Currently henchman uses SSH as a transport to execute a plan on hosts specified in an inventory. It will be possible to use custom transports and custom inventory scripts in the future. 

## Building Henchman
* Install go [here](https://golang.org/doc/install)
* Clone this repo to `$GOPATH/src/github.com/apigee`
* If you have not done so, set `export $PATH=$PATH:$GOPATH/bin`
* `go get github.com/tools/godep`
* `godep restore`
* `go build -o bin/henchman`

## Contributing
Just clone or fork from [https://github.com/apigee/henchman](https://github.com/apigee/henchman) and off you go! Fixing issues marked `easy` in the issue tracker is a great way to get started. Or you can help by creating more modules.  Look at the modules section for more details.
