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

type TemplateModule struct {
	Owner string
	Group string
	Mode  string
	Dest  string

	// is ".henchman/(folder/file name)
	RmtSrc   string
	Override string
}

var result map[string]interface{} = map[string]interface{}{}
var templateParams TemplateModule = TemplateModule{}

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
	if err := dec.Decode(&templateParams); err != nil {
		panic(err.Error())
	}

	if templateParams.Dest == "" {
		panic("Required parameter 'dest' not found")
	}

	// Override param is passed in as a string and bools tend to default to false
	override := true
	str := strings.ToLower(templateParams.Override)
	if str == "false" {
		override = false
	} else if str != "" && str != "true" {
		panic("override param must be true or false")
	}

	// sets parameters of copy module as output
	result["output"] = map[string]interface{}{
		"override": templateParams.Override,
		"dest":     templateParams.Dest,
		"owner":    templateParams.Owner,
		"group":    templateParams.Group,
		"mode":     templateParams.Mode,
	}

	result["status"] = "changed"
	msg := fmt.Sprintf("Success file/folder copied to %s.", templateParams.Dest)

	srcInfo, _ := os.Stat(templateParams.RmtSrc)
	destInfo, err := os.Stat(templateParams.Dest)
	if err != nil && !os.IsNotExist(err) {
		panic(fmt.Sprintf("Dest exists - %s", err.Error()))
	} else if os.IsNotExist(err) {
		// Creates necessary file/folder basically a mkdir -P call
		if err := os.MkdirAll(templateParams.Dest, 0755); err != nil {
			panic(fmt.Sprintf("Error creating directories - %s", err.Error()))
		}

		if err := MoveSrcToDest(templateParams.RmtSrc, templateParams.Dest); err != nil {
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
				if err := MoveSrcToDest(templateParams.RmtSrc, templateParams.Dest); err != nil {
					panic(err.Error())
				}
			} else {
				result["status"] = "ok"
				msg = fmt.Sprintf("File/folder not copied, %s already exists", templateParams.Dest)
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
	return filepath.Walk(templateParams.RmtSrc,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			element := strings.TrimPrefix(path, templateParams.RmtSrc)
			if element == "" {
				return fmt.Errorf("root: %s, path: %s", templateParams.RmtSrc, path)
			}

			destPath := filepath.Join(templateParams.Dest, element)
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
	if templateParams.Mode != "" {
		i, err := strconv.ParseInt(templateParams.Mode, 8, 32)
		if err != nil {
			return fmt.Errorf("Error retrieving mode - %s", err.Error())
		}
		if i < 0 {
			return fmt.Errorf("Error mode must be an unsigned integer")
		}

		if err := os.Chmod(templateParams.Dest, os.FileMode(i)); err != nil {
			return fmt.Errorf("Error chmod file/folder - %s", err.Error())
		}
	}

	return nil
}

// Sets ownership of file/folder using /bin/chown
func SetOwner() error {
	// using chown command os.Chown(...) only takes in ints for now GO 1.5
	var cmd *exec.Cmd
	cmdList := []string{"-R", templateParams.Owner + ":" + templateParams.Group, templateParams.Dest}
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
