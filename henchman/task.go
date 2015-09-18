package henchman

// FIXME: Use reflection to figure out what fields the Task
// export. But for now, hardcode the field names
var TaskFields = map[string]bool{
	"name":          true,
	"ignore_errors": true,
	"local":         true,
	"when":          true,
	"register":      true,
}

type Task struct {
	Id           string
	Name         string
	Module       *Module
	IgnoreErrors bool `yaml:"ignore_errors"`
}
