package henchman

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type Configuration struct {
	Log string
}

func InitConfiguration() error {
	buf, err := ioutil.ReadFile("conf.yaml")
	if err != nil {
		return fmt.Errorf("Error reading conf.yaml :: " + err.Error())
	}

	var config Configuration
	err = yaml.Unmarshal(buf, &config)
	if err != nil {
		return fmt.Errorf("Error unmarshalling conf.yaml :: " + err.Error())
	}

	LogFile = config.Log

	return nil
}
