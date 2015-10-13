package henchman

import (
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

func (plan *Plan) Execute() error {
	machines := plan.Inventory.Machines()
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
				task.Vars["current_host"] = machine
				err := task.Render(registerMap)
				if err != nil {
					log.Printf("Error Rendering Task: %v.  Received: %v\n", task.Name, err.Error())
					return
				}
				taskResult, err := task.Run(machine, registerMap)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println(taskResult.Output)
				log.Println(registerMap)
			}
		}()
	}
	wg.Wait()
	return nil
}
