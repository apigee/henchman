package henchman

import (
	"bytes"
	"encoding/json"
	"log"
)

type TestTransport struct{}

func (tt *TestTransport) Initialize(config *TransportConfig) error {
	log.Printf("Initialized test transport\n")
	return nil
}

func (tt *TestTransport) Exec(cmd string, stdin []byte, sudo bool) (*bytes.Buffer, error) {
	// Create a dummy map mirroring the task result.
	// We're not using TaskResult directly since changes in that contract
	// will be caught for free
	taskResult := map[string]string{
		"status": "true",
		"output": "foo",
		"msg":    "the dude abides",
	}
	jsonified, err := json.Marshal(taskResult)
	if err != nil {
		log.Fatalf(err.Error())
	}
	return bytes.NewBuffer([]byte(jsonified)), nil
}

func (tt *TestTransport) Put(source, destination string) error {
	log.Printf("Transfered from %s to %s\n", source, destination)
	return nil
}
