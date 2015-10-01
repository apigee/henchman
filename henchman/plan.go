package henchman

import (
	"log"
	"sync"
)

type Plan struct {
	Name      string
	Inventory Inventory
	Vars      TaskVars
	Tasks     []*Task
}

func (plan *Plan) Execute() error {
	machines := plan.Inventory.Machines()
	log.Printf("Will attempt to execute the plan on %d machines\n", len(machines))
	// FIXME: Don't use localhost
	wg := new(sync.WaitGroup)
	for _, _machine := range machines {
		wg.Add(1)
		machine := _machine
		go func() {
			defer wg.Done()
			for _, task := range plan.Tasks {
				taskResult, err := task.Run(machine)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println(taskResult)

			}
		}()
	}
	wg.Wait()
	return nil
}
