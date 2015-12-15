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

func InitConfiguration(filename string) error {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("Error reading conf.json :: " + err.Error())
	}

	err = json.Unmarshal(buf, &Config)
	if err != nil {
		return fmt.Errorf("Error unmarshalling conf.yaml :: " + err.Error())
	}

	return nil
}
