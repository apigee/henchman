package henchman

import (
	"fmt"
	"log"
	"sync"

	"github.com/mgutz/ansi"
)

// For ANSI color codes
var statuses = map[string]string{
	"reset":   ansi.ColorCode("reset"),
	"ok":      ansi.ColorCode("green"),
	"changed": ansi.ColorCode("yellow"),
	"failure": ansi.ColorCode("red"),
	"error":   ansi.ColorCode("red"),
	"ignored": ansi.ColorCode("cyan"),
}

type VarsMap map[interface{}]interface{}
type RegMap map[string]interface{}

type Plan struct {
	Name      string
	Inventory Inventory
	Vars      VarsMap
	Tasks     []*Task
}

const HENCHMAN_PREFIX = "henchman_"

func localhost() *Machine {
	tc := make(TransportConfig)
	local, _ := NewLocal(&tc)
	localhost := Machine{}
	localhost.Hostname = "127.0.0.1"
	localhost.Transport = local
	return &localhost
}

func (plan *Plan) Execute(machines []*Machine) error {
	local := localhost()

	log.Printf("Executing plan `%s' on %d machines\n", plan.Name, len(machines))

	resetCode := statuses["reset"]
	wg := new(sync.WaitGroup)
	for _, _machine := range machines {
		machine := _machine
		wg.Add(1)
		//		machineVars := plan.Inventory.Groups[machine.Group].Vars
		// NOTE: need individual registerMap for each machine
		registerMap := make(RegMap)
		go func() {
			defer wg.Done()
			var actualMachine *Machine
			for _, task := range plan.Tasks {
				// copy of task.Vars. It'll be different for each machine
				vars := make(VarsMap)
				MergeMap(plan.Vars, vars, true)
				MergeMap(machine.Vars, vars, true)
				MergeMap(task.Vars, vars, true)
				if task.Local == true {
					actualMachine = local
				} else {
					actualMachine = machine
				}

				vars["current_host"] = actualMachine.Hostname
				err := task.Render(vars, registerMap)

				if err != nil {
					log.Printf("Error Rendering Task: %v.  Received: %v\n", task.Name, err.Error())
					return
				}
				taskResult, err := task.Run(actualMachine, vars, registerMap)
				if err != nil {
					log.Println(err)
					return
				}
				colorCode := statuses[taskResult.State]
				fmt.Printf("%s[%s]: %s - %s]\n", colorCode, actualMachine.Hostname, taskResult.State)
				// print only when --debug is on
				fmt.Printf("%s[%s]: Task: \"%s\" Output - %v", colorCode, actualMachine.Hostname, task.Name, taskResult.Output)
				fmt.Print("%s\n", resetCode)
				if (taskResult.State == "error" || taskResult.State == "failure") && (!task.IgnoreErrors) {
					break
				}
			}
		}()
	}
	wg.Wait()
	return nil
}
