package henchman

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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
    "copy": ["stage", "exec_module"],
    "template": ["process_template", "stage", "reset_src", "exec_module"]
  }
}
`

// FIXME: If a custom configuration is given, it should be merged with DEFAULT configuration
func InitConfiguration(filename string) error {
	var buf []byte
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		fmt.Printf("conf.json is not present. Applying default configuration\n")
		buf = []byte(DEFAULT_CONFIGURATION)
	} else {
		buf, err := ioutil.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("Error reading conf.json :: " + err.Error())
		}
	}

	err := json.Unmarshal(buf, &Config)
	if err != nil {
		return fmt.Errorf("Error unmarshalling conf.json :: " + err.Error())
	}

	return nil
}
