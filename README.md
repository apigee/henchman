

# henchman [![Build Status](https://travis-ci.org/apigee/henchman.svg?branch=master)](https://travis-ci.org/apigee/henchman)
## What is Henchman
Henchman is a non-agent based orchestration and automation tool created in Go, and inspired by Ansible.

## Why Go?
Although, python and ruby are awesome as systems languages, Golang fits this 'get shit done' niche more.  Dependencies is generally an issue in ruby/python where users have to install and maintain gems/pip modules.  With Go there are no dependencies to install, and a single binary file and can shipped and ready to use.  In addition, Go has built in concurrency constructs, and a specific way to do things.

## Setup
* Install go [here](https://golang.org/doc/install)
* Clone this repo to `$GOPATH/src/github.com/apigee`
* If you have not done so, set `export $PATH=$PATH:$GOPATH/bin`
* `go get github.com/tools/godep`
* `godep restore`
* `go build -o bin/henchman`

## How Henchman Works
Henchman executes a plan (a collection of tasks) on a given set of machines.  Each plan will run the list of tasks by SSHing into a machine, and generally copy over the module specified by the task and execute it.  

## Plan
A plan is a collection of tasks written in YAML notation.

Insert a fat Plan example here with all features
Insert pratical use case here

### Hosts
Explain features of hosts section.  Include Examples

### Vars
Explain all aspects and features of Vars here.  Include Examples

### Tasks
Explain all features of tasks.  Include Examples

## Inventories
Inventory takes 2 keys at the top: 'groups' and 'hostvars'.
Under 'groups', you can specify various group names and they in turn can have 'hosts' and 'vars' applicable to the group of hosts
'hostvars' allow overrides at individual host level

groups:
  all:
    hosts:
      - 192.168.33.10
    vars:
      test: 10
 
  zookeeper:
    hosts:
      - z1
      - z2
      - z3
hostvars:
   "192.168.33.10":
      test: 20


## How to Use
CLI commands insert here.

If you are using Vagrant to spin up vms
`bin/henchman exec plan.yml --inventory inv.yaml --user vagrant --keyfile <  >`

## Modules

## Contributing

Just clone or fork from [https://github.com/apigee/henchman](https://github.com/apigee/henchman) and off you go!
Or you can help by creating more modules.  Look at the modules section for more details.
