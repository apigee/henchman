package henchman

import (
	//"encoding/json"
	"fmt"
	log "gopkg.in/Sirupsen/logrus.v0"
	//"github.com/kr/pretty"
	"io/ioutil"
	"os"
	"path"
	"reflect"
)

// source values will override dest values if override is true
// else dest values will not be overridden
func MergeMap(src map[interface{}]interface{}, dst map[interface{}]interface{}, override bool) {
	for variable, value := range src {
		if override == true {
			dst[variable] = value
		} else if _, present := dst[variable]; !present {
			dst[variable] = value
		}
	}
}

// used to make tmp files in *_test.go
func createTempDir(folder string) string {
	name, _ := ioutil.TempDir("/tmp", folder)
	return name
}

func writeTempFile(buf []byte, fname string) string {
	fpath := path.Join("/tmp", fname)
	ioutil.WriteFile(fpath, buf, 0644)
	return fpath
}

func rmTempFile(fpath string) {
	os.Remove(fpath)
}

// recursively print a map.  Only issue is everything is out of order in a map.  Still prints nicely though
func printRecurse(output interface{}, padding string, retVal string) string {
	tmpVal := retVal
	switch output.(type) {
	case map[string]interface{}:
		for key, val := range output.(map[string]interface{}) {
			switch val.(type) {
			case map[string]interface{}:
				tmpVal += fmt.Sprintf("%s%v:\n", padding, key)
				//log.Debug("%s%v:\n", padding, key)
				tmpVal += printRecurse(val, padding+"  ", "")
			default:
				tmpVal += fmt.Sprintf("%s%v: %v (%v)\n", padding, key, val, reflect.TypeOf(val))
				//log.Debug("%s%v: %v\n", padding, key, val)
			}
		}
	default:
		tmpVal += fmt.Sprintf("%s%v (%s)\n", padding, output, reflect.TypeOf(output))
		//log.Debug("%s%v\n", padding, output)
	}

	return tmpVal
}

func printOutput(taskName string, output interface{}) {
	log.WithFields(log.Fields{
		"name":   taskName,
		"output": printRecurse(output, "", "\n"),
	}).Debug("Task Output")
	/*
		switch output.(type) {
		default:
			convOutput, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				fmt.Errorf("Error printing output - %s", err.Error())
			}
			log.Debug(string(convOutput))
		}
	*/
}

/*
func printTask(task *Task, output interface{}) {
	fmt.Printf("Task: \"%s\"\n", task.Name)
	fmt.Println("Output: \n--------------------")

	val, ok := task.Module.Params["loglevel"]
	if ok && val == "debug" {
		fmt.Printf("% v\n", pretty.Formatter(output))
	} else {
		fmt.Printf("%# v\n", pretty.Formatter(output))
	}
}
*/
