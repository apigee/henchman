package henchman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type Configuration struct {
	Log       string
	ExecOrder map[string][]string
}

// Global config object for henchman to use
var Config Configuration

const DEFAULT_CONFIGURATION = `
{
  "log": "~/.henchman/system.log",
  "execOrder": {
    "default": ["exec_module"],
    "copy": ["put_for_copy", "copy_remote", "exec_module"],
    "template": ["process_template", "put_for_copy", "copy_remote", "reset_src", "exec_module"]
  }
}
`

// FIXME: If a custom configuration is given, it should be merged with DEFAULT configuration
func InitConfiguration(filename string) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Printf("Error reading conf.json. Applying default configuration\n")
		buf = []byte(DEFAULT_CONFIGURATION)
	}

	err = json.Unmarshal(buf, &Config)
	if err != nil {
		return fmt.Errorf("Error unmarshalling conf.yaml :: " + err.Error())
	}

	return nil
}
