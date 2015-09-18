package henchman

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
