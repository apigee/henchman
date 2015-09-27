package main

import (
	"github.com/apigee/henchman/henchman"
)

func main() {
	tc := make(henchman.TransportConfig)
	tc["username"] = "vagrant"
	tc["keyfile"] = "/Users/sudharsh/.vagrant.d/insecure_private_key"
	tc["hostname"] = "10.224.192.11"

	ssht, _ := henchman.NewSSH(&tc)

	machine := henchman.Machine{}
	machine.Hostname = "router"
	machine.Transport = ssht

	task := henchman.Task{}
	task.Name = "Check it out"
	task.Module, _ = henchman.NewModule("yum", "package=nginx")
	task.Sudo = true
	task.Run(&machine)
}
