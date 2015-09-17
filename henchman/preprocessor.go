package henchman

import (
	_ "errors"
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

func PreprocessTasks(taskSection []*TaskProxy) ([]*Task, error) {
	//var tasks []TaskProxy
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
			log.Printf("Reading %s\n", tp.Include)
			log.Printf("Got %d tasks\n", len(innerPlan.Tasks))
			log.Printf(string(buf))
			tasks = append(tasks, innerPlan.Tasks...)
		} else {
			task.Name = tp.Name
			tasks = append(tasks, &task)
		}
	}
	return tasks, nil
}

func PreprocessPlan(buf []byte) (*Plan, error) {
	var px PlanProxy
	err := yaml.Unmarshal(buf, &px)
	plan := Plan{}

	tasks, err := PreprocessTasks(px.TaskProxies)
	if err != nil {
		log.Printf("Error processing tasks\n")
		return nil, err
	}
	plan.Tasks = tasks
	return &plan, err
}
