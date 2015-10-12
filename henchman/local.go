package henchman

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Transport for the current machine on which henchman is being run on
type LocalTransport struct{}

func (local *LocalTransport) Initialize(config *TransportConfig) error {
	return nil
}

func (local *LocalTransport) Exec(cmdStr string, stdin []byte, sudoEnabled bool) (*bytes.Buffer, error) {
	var b bytes.Buffer
	if sudoEnabled {
		cmdStr = fmt.Sprintf("/bin/bash -c 'sudo -H -u root %s'", cmdStr)
	}
	// FIXME: This is kinda dumb and can break for weird inputs. Make this more robust
	commands := strings.Split(cmdStr, " ")
	cmd := exec.Command(commands[0], commands[1:]...)
	// We need to setup two sets of cmds for piping stdin into the command that
	// has to be executed
	stdinPipe := exec.Command("echo", string(stdin))
	var err error
	cmd.Stdin, err = stdinPipe.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stdout = &b
	cmd.Stderr = &b

	err = cmd.Start()
	if err != nil {
		return nil, err
	}
	err = stdinPipe.Run()
	if err != nil {
		return nil, err
	}
	err = cmd.Wait()
	if err != nil {
		return nil, err
	}
	return &b, err
}

func (local *LocalTransport) Put(source, destination string, dstType string) error {
	return nil
}
