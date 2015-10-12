

# henchman [![Circle CI](https://circleci.com/gh/apigee/henchman/tree/master.svg?style=svg)](https://circleci.com/gh/apigee/henchman/tree/master)
## What is Henchman
Henchman is a non-agent based orchestration and automation tool created in Go, and inspired by Ansible.

## Why Go?
Although, python and ruby are awesome as systems languages, runtime dependencies are generally an issue where users have to install and maintain gems/pip modules. System tools should never enforce the headache of having to manage multiple runtime dependencies and as the number of machines managed increases, it becomes an additional overhead to take care of. With Golang there is no such overhead and a single static binary file can be shipped for use.

## Why are modules written in Python then?
Henchman allows the modules to be written in any programming language as long as they have access to STDOUT and STDIN.  In the future, we may look into using just bash scripts for the modules if Python becomes the bottleneck. For now, writing python modules gives us a nice balance between speed of development and not having to enforce dependencies. The modules themselves use only system libraries.

## Building Henchman
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

## How to Use
CLI commands insert here.

If you are using Vagrant to spin up vms
`bin/henchman exec plan.yml --inventory inv.yaml --user vagrant --keyfile <  >`

## Modules

## Contributing

Just clone or fork from [https://github.com/apigee/henchman](https://github.com/apigee/henchman) and off you go!
Or you can help by creating more modules.  Look at the modules section for more details.
