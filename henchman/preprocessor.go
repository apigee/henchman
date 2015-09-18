package henchman

import (
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
)

type PlanProxy struct {
	TaskProxies []*TaskProxy `yaml:"tasks"`
}

type TaskProxy struct {
	Task    `yaml:",inline"`
	Include string
}

func (tp *TaskProxy) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var tmap map[string]interface{}
	err := unmarshal(&tmap)
	if err != nil {
		return err
	}
	for field, val := range tmap {
		switch field {
		case "name":
			tp.Name = val.(string)
		case "ignore_errors":
			tp.IgnoreErrors = val.(bool)
		case "include":
			tp.Include = val.(string)
		default:
			// We have a module
			module, err := NewModule(field, val.(string))
			if err != nil {
				return err
			}
			tp.Module = module
		}
	}
	return nil
}

func PreprocessTasks(taskSection []*TaskProxy) ([]*Task, error) {
	var tasks []*Task
	for _, tp := range taskSection {
		task := Task{}
		if tp.Include != "" {
			// FIXME: resolve if templating is found
			buf, err := ioutil.ReadFile(tp.Include)
			if err != nil {
				return nil, err
			}
			innerPlan, err := PreprocessPlan(buf)
			if err != nil {
				return nil, err
			}
			tasks = append(tasks, innerPlan.Tasks...)
		} else {
			task.Name = tp.Name
			if tp.Module == nil {
				return nil, errors.New("This task doesn't have a valid module")
			}
			task.Module = tp.Module
			task.IgnoreErrors = tp.IgnoreErrors
			tasks = append(tasks, &task)
		}
	}
	return tasks, nil
}

func PreprocessPlan(buf []byte) (*Plan, error) {
	var px PlanProxy
	err := yaml.Unmarshal(buf, &px)
	if err != nil {
		return nil, err
	}
	plan := Plan{}
	tasks, err := PreprocessTasks(px.TaskProxies)
	if err != nil {
		log.Printf("Error processing tasks\n")
		return nil, err
	}
	plan.Tasks = tasks
	return &plan, err
}
