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

func getMachinesFromInventory(inv Inventory, tc TransportConfig) ([]*Machine, error) {
	var machines []*Machine
	machineSet := make(map[string]bool)
	for _, hostGroup := range inv.Groups {
		for _, hostname := range hostGroup.Hosts {
			if _, present := machineSet[hostname]; !present {
				machineSet[hostname] = true
				machine := &Machine{}
				machine.Hostname = hostname
				tcCurr := make(TransportConfig)
				tcCurr["hostname"] = hostname
				for k, v := range tc {
					tcCurr[k] = v
				}
				machine.Vars = hostGroup.Vars
				ssht, err := NewSSH(&tcCurr)
				if err != nil {
					return nil, err
				}
				machine.Transport = ssht
				machines = append(machines, machine)
			}
		}
	}
	return machines, nil
}

func (plan *Plan) Execute(tc TransportConfig) error {
	machines, err := getMachinesFromInventory(plan.Inventory, tc)
	if err != nil {
		return err
	}
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
				// copy of task.Vars. It'll be different for each machine
				vars := make(VarsMap)
				for k, v := range task.Vars {
					vars[k] = v
				}
				vars["current_host"] = machine
				MergeMap(machine.Vars, vars, true)
				plan.Inventory.MergeHostVars(machine.Hostname, vars)
				err := task.Render(vars, registerMap)
				if err != nil {
					log.Printf("Error Rendering Task: %v.  Received: %v\n", task.Name, err.Error())
					return
				}
				taskResult, err := task.Run(machine, vars, registerMap)
				if err != nil {
					log.Println(err)
					return
				}
				log.Println(taskResult.Output)
			}
		}()
	}
	wg.Wait()
	return nil
}
