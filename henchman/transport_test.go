package henchman

import (
	"bytes"
	"log"
)

type TestTransport struct{}

func (tt *TestTransport) Initialize(config *TransportConfig) error {
	log.Printf("Initialized test transport\n")
	return nil
}

func (tt *TestTransport) Exec(cmd string, stdin []byte) (*bytes.Buffer, error) {
	return bytes.NewBuffer([]byte("cmd " + cmd + " executed")), nil
}

func (tt *TestTransport) Put(source, destination string) error {
	log.Printf("Transfered from %s to %s\n", source, destination)
	return nil
}
