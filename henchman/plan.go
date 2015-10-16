package henchman

import (
	//"fmt"
	"log"
	"sync"
)

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

	// FIXME: Don't use localhost
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

				log.Println(taskResult)
				/*
					fmt.Printf("State: %v\n", taskResult.State)
					fmt.Printf("Msg: %v\n", taskResult.Msg)
					fmt.Printf("Output: %v\n", taskResult.Output)
				*/
			}
		}()
	}
	wg.Wait()
	return nil
}
