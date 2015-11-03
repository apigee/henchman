package henchman

import (
	log "gopkg.in/Sirupsen/logrus.v0"
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

	log.WithFields(log.Fields{
		"plan":         plan.Name,
		"num machines": len(machines),
	}).Info("Executing plan")

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
					log.WithFields(log.Fields{
						"task":  task.Name,
						"host":  actualMachine.Hostname,
						"error": err.Error(),
					}).Error("Error Rendering Task")

					return
				}

				log.WithFields(log.Fields{
					"task": task.Name,
					"host": actualMachine.Hostname,
				}).Info("Starting Task")

				taskResult, err := task.Run(actualMachine, vars, registerMap)
				if err != nil {
					log.WithFields(log.Fields{
						"task":  task.Name,
						"host":  actualMachine.Hostname,
						"error": err.Error(),
					}).Error("Error Running Task")

					return
				}

				colorCode := statuses[taskResult.State]

				//NOTE: make a color code create function
				log.WithFields(log.Fields{
					"task":  task.Name,
					"host":  actualMachine.Hostname,
					"state": colorCode + taskResult.State + resetCode,
					"msg":   taskResult.Msg,
				}).Info("Task Complete")

				// print only when --debug is on
				if Debug {
					printOutput(task.Name, taskResult.Output)
				}

				if (taskResult.State == "error" || taskResult.State == "failure") && (!task.IgnoreErrors) {
					break
				}
			}
		}()
	}
	wg.Wait()
	return nil
}
