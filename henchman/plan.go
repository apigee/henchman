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

func localhost() *Machine {
	tc := make(TransportConfig)
	local, _ := NewLocal(&tc)
	localhost := Machine{}
	localhost.Hostname = "127.0.0.1"
	localhost.Transport = local
	return &localhost
}

func (plan *Plan) Execute() error {
	machines := plan.Inventory.Machines()
	local := localhost()

	log.Printf("Executing plan `%s' on %d machines\n", plan.Name, len(machines))
	// FIXME: Don't use localhost
	wg := new(sync.WaitGroup)
	for _, _machine := range machines {
		wg.Add(1)
		machine := _machine
		// NOTE: need individual registerMap for each machine
		registerMap := make(RegMap)
		go func() {
			defer wg.Done()
			for _, task := range plan.Tasks {
				if task.Local == true {
					task.Vars["current_host"] = local.Hostname
				} else {
					task.Vars["current_host"] = machine.Hostname
				}

				err := task.Render(registerMap)
				if err != nil {
					log.Printf("Error Rendering Task: %v.  Received: %v\n", task.Name, err.Error())
					return
				}

				var taskResult *TaskResult
				if task.Local == true {
					taskResult, err = task.Run(local, registerMap)
				} else {
					taskResult, err = task.Run(machine, registerMap)
				}
				if err != nil {
					log.Println(err)
					return
				}

				log.Println(taskResult.Output)
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
