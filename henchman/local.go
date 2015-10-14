package henchman

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"

	"github.com/flynn/go-shlex"
)

// Transport for the current machine on which henchman is being run on
type LocalTransport struct{}

func (local *LocalTransport) Initialize(config *TransportConfig) error {
	return nil
}

func (local *LocalTransport) Exec(cmdStr string, stdin []byte, sudoEnabled bool) (*bytes.Buffer, error) {
	var b bytes.Buffer
	var err error

	cmdStr = strings.Replace(cmdStr, "\"", "\\\"", -1)
	if sudoEnabled {
		cmdStr = fmt.Sprintf("/bin/sh -c \"sudo -H -u root %s\"", cmdStr)
	} else {
		cmdStr = fmt.Sprintf("/bin/sh -c \"%s\"", cmdStr)
	}
	// FIXME: This is kinda dumb and can break for weird inputs. Make this more robust
	commands, err := shlex.Split(cmdStr)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(commands[0], commands[1:]...)
	// We need to setup two sets of cmds for piping stdin into the command that
	// has to be executed
	var stdinPipe *exec.Cmd
	if stdin != nil {
		stdinPipe = exec.Command("echo", string(stdin))
		cmd.Stdin, err = stdinPipe.StdoutPipe()
		if err != nil {
			return nil, err
		}
	}
	cmd.Stdout = &b
	cmd.Stderr = &b
	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	if stdinPipe != nil {
		err = stdinPipe.Run()
		if err != nil {
			return nil, err
		}
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	return &b, err
}

func (local *LocalTransport) Put(source, destination string, _ string) error {
	// Run cp in a subshell to expand env variables.
	cpCmd := fmt.Sprintf("cp -r \"%s\" \"%s\"", source, destination)
	_, err := local.Exec(cpCmd, nil, false)
	return err
}

func NewLocal(config *TransportConfig) (*LocalTransport, error) {
	local := LocalTransport{}
	return &local, local.Initialize(config)
}
