package main

import (
	_ "bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"encoding/json"
)

type CopyModule struct {
	Owner string
	Group string
	Mode  string
	Dest  string

	// is ".henchman/(folder/file name)
	RmtSrc   string
	Override string
}

var result map[string]interface{} = map[string]interface{}{}
var copyParams CopyModule = CopyModule{}

func main() {
	// recover code
	// also does the printout of result
	defer func() {
		if r := recover(); r != nil {
			result["status"] = "error"
			result["msg"] = r
		}

		output, err := json.Marshal(result)
		if err != nil {
			panic(err)
		}
		fmt.Print(string(output))
	}()

	// basically unmarshall but can take in a io.Reader
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&copyParams); err != nil {
		panic(err.Error())
	}

	if copyParams.Dest == "" {
		panic("Required parameter 'dest' not found")
	}

	// Override param is passed in as a string and bools tend to default to false
	override := true
	str := strings.ToLower(copyParams.Override)
	if str == "false" {
		override = false
	} else if str != "" && str != "true" {
		panic("override param must be true or false")
	}

	// sets parameters of copy module as output
	result["output"] = map[string]interface{}{
		"override": copyParams.Override,
		"dest":     copyParams.Dest,
		"owner":    copyParams.Owner,
		"group":    copyParams.Group,
		"mode":     copyParams.Mode,
	}

	result["status"] = "changed"
	msg := fmt.Sprintf("Success file/folder copied to %s.", copyParams.Dest)

	srcInfo, _ := os.Stat(copyParams.RmtSrc)
	destInfo, err := os.Stat(copyParams.Dest)
	if err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("Dest exists - %s", err.Error()))
	} else if os.IsNotExist(err) {
		// Creates necessary file/folder basically a mkdir -P call
		if err := os.MkdirAll(copyParams.Dest, 0755); err != nil {
			panic(fmt.Sprintf("Error creating directories - %s", err.Error()))
		}

		if err := MoveSrcToDest(copyParams.RmtSrc, copyParams.Dest); err != nil {
			panic(err.Error())
		}
	} else {
		if srcInfo.IsDir() && destInfo.IsDir() {
			if err := MergeFolders(override); err != nil {
				panic(err.Error())
			}

			msg = "Success folders merged."
			if override {
				msg += " Existing copies of dest files overwritten."
			} else {
				msg += " Existing copies of dest preserved."
			}
		} else {
			if override {
				if err := MoveSrcToDest(copyParams.RmtSrc, copyParams.Dest); err != nil {
					panic(err.Error())
				}
			} else {
				result["status"] = "ok"
				msg = fmt.Sprintf("File/folder not copied, %s already exists", copyParams.Dest)
			}
		}
	}

	if err := SetOwner(); err != nil {
		panic(err.Error())
	}
	if err := SetMode(); err != nil {
		panic(err.Error())
	}

	result["msg"] = msg
}

// Merges two folders based off override parameter value
func MergeFolders(override bool) error {
	return filepath.Walk(copyParams.RmtSrc,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			element := strings.TrimPrefix(path, copyParams.RmtSrc)
			if element == "" {
				return fmt.Errorf("root: %s, path: %s", copyParams.RmtSrc, path)
			}

			destPath := filepath.Join(copyParams.Dest, element)
			destInfo, err := os.Stat(destPath)

			if err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("At %s - %s", destPath, err.Error())
			} else if os.IsNotExist(err) || (!destInfo.IsDir() && override) {
				MoveSrcToDest(path, destPath)
			}

			return nil
		})
}

// Sets file/folder permissions
func SetMode() error {
	if copyParams.Mode != "" {
		i, err := strconv.ParseInt(copyParams.Mode, 8, 32)
		if err != nil {
			return fmt.Errorf("Error retrieving mode - %s", err.Error())
		}
		if i < 0 {
			return fmt.Errorf("Error mode must be an unsigned integer")
		}

		if err := os.Chmod(copyParams.Dest, os.FileMode(i)); err != nil {
			return fmt.Errorf("Error chmod file/folder - %s", err.Error())
		}
	}

	return nil
}

// Sets ownership of file/folder using /bin/chown
func SetOwner() error {
	// using chown command os.Chown(...) only takes in ints for now GO 1.5
	var cmd *exec.Cmd
	cmdList := []string{"-R", copyParams.Owner + ":" + copyParams.Group, copyParams.Dest}
	cmd = exec.Command("/bin/chown", cmdList...)

	if output, err := cmd.CombinedOutput(); err != nil {
		fmt.Errorf("Error chown file/folder - %s", string(output))
	}

	return nil
}

// Moves RmtSrc file/folder to Dest if the dest exists it removes it
func MoveSrcToDest(src, dest string) error {
	// Removes the last file/folder
	if err := os.RemoveAll(dest); err != nil {
		if !os.IsExist(err) {
			return fmt.Errorf("Error removing endpoint for override - %s", err.Error())
		}
	}

	// Moves the src to dest
	if err := os.Rename(src, dest); err != nil {
		return fmt.Errorf("Error copying file/folder - %s", err.Error())
	}

	return nil
}
