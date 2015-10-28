package henchman

import (
	//"encoding/json"
	"fmt"
	//"github.com/kr/pretty"
	"io/ioutil"
	"os"
	"path"
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
func printRecurse(output interface{}, padding string) {
	switch output.(type) {
	case map[string]interface{}:
		for key, val := range output.(map[string]interface{}) {
			switch val.(type) {
			case map[string]interface{}:
				fmt.Printf("%s%v:\n", padding, key)
				printRecurse(val, padding+"  ")
			default:
				fmt.Printf("%s%v: %v\n", padding, key, val)
			}
		}
	default:
		fmt.Printf("%s%v\n", padding, output)
	}
}

func printOutput(taskName string, output interface{}) {
	fmt.Printf("Task: \"%s\"\n", taskName)
	fmt.Println("Output: \n--------------------")
	/*
		switch output.(type) {
		default:
			convOutput, err := json.MarshalIndent(output, "", "  ")
			if err != nil {
				fmt.Errorf("Error printing output - %s", err.Error())
			}
			fmt.Printf(string(convOutput))
		}
	*/

	printRecurse(output, "")
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
