package henchman

import (
	"encoding/json"
	"fmt"
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

func printOutput(coloCode string, hostname string, taskName string, output interface{}) error {
	fmt.Printf("Task: \"%s\"\n", taskName)
	fmt.Println("Output: ")
	switch output.(type) {
	default:
		convOutput, err := json.MarshalIndent(output, "", "  ")
		if err != nil {
			fmt.Errorf("Error printing output - %s", err.Error())
		}
		fmt.Println(string(convOutput))
	}

	return nil
}
