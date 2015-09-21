package henchman

import (
	"fmt"
	"log"
	"path"

	"code.google.com/p/go-uuid/uuid"
)

type TaskVars map[interface{}]interface{}

type Task struct {
	Id           string
	Name         string
	Module       *Module
	IgnoreErrors bool `yaml:"ignore_errors"`
	Local        bool
	When         string
	Register     string
	Vars         TaskVars
}

func (task *Task) Run(machine *Machine) error {
	// Resolving module path
	task.Id = uuid.New()
	modPath, err := task.Module.Resolve()
	if err != nil {
		return err
	}
	// Transfering the module
	remoteModDir := path.Join("$HOME/.henchman", task.Id)
	remoteModPath := path.Join(remoteModDir, task.Module.Name)

	// Create the remoteModDir
	_, err = machine.Transport.Exec(fmt.Sprintf("mkdir -p %s\n", remoteModDir))
	if err != nil {
		log.Printf("Error while creating remote module path\n")
		return err
	}

	// Put the module on the remotePath
	err = machine.Transport.Put(modPath, remoteModDir)
	if err != nil {
		return err
	}
	// Executing the module
	buf, err := machine.Transport.Exec(remoteModPath)
	if err != nil {
		return err
	}
	log.Printf("%s\n", buf)
	return nil
}
