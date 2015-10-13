package henchman

import (
	"log"
	"strings"
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

func GetHenchmanVars(vars VarsMap) VarsMap {
	var henchmanVars VarsMap

	for k, v := range vars {
		if strings.Contains(k.(string), HENCHMAN_PREFIX) {
			if len(henchmanVars) == 0 {
				henchmanVars = make(VarsMap)
			}
			parts := strings.Split(k.(string), HENCHMAN_PREFIX)
			henchmanVars[parts[len(parts)-1]] = v
		}
	}

	return henchmanVars
}

func mergeHenchmanVarsWithTransport(tcCurr TransportConfig, hostname string, hostGroupVars VarsMap,
	hostVars map[string]map[interface{}]interface{}) {

	henchmanVars := make(VarsMap)
	henchmanVars = GetHenchmanVars(hostGroupVars)
	for k, v := range henchmanVars {
		tcCurr[k.(string)] = v.(string)
	}
	if _, present := hostVars[hostname]; present {
		henchmanVars = GetHenchmanVars(hostVars[hostname])
		for k, v := range henchmanVars {
			tcCurr[k.(string)] = v.(string)
		}
	}
}

func getMachinesFromInventory(inv Inventory, tc TransportConfig) ([]*Machine, error) {
	var machines []*Machine
	machineSet := make(map[string]bool)
	for group, hostGroup := range inv.Groups {
		for _, hostname := range hostGroup.Hosts {
			if _, present := machineSet[hostname]; !present {
				machineSet[hostname] = true
				machine := &Machine{}
				machine.Hostname = hostname
				machine.Group = group
				tcCurr := make(TransportConfig)
				tcCurr["hostname"] = hostname
				for k, v := range tc {
					tcCurr[k] = v
				}
				mergeHenchmanVarsWithTransport(tcCurr, hostname, hostGroup.Vars, inv.HostVars)

				log.Println("Transport Config for machine", hostname, ": ", tcCurr)
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
		machine := _machine
		wg.Add(1)
		machineVars := plan.Inventory.Groups[machine.Group].Vars
		// NOTE: need individual registerMap for each machine
		registerMap := make(RegMap)
		go func() {
			defer wg.Done()
			for _, task := range plan.Tasks {
				// copy of task.Vars. It'll be different for each machine
				vars := make(VarsMap)
				MergeMap(task.Vars, vars, true)
				MergeMap(machineVars, vars, true)
				vars["current_host"] = machine
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
