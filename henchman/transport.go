package henchman

import (
	"bytes"
)

type TransportConfig map[string]string

type TransportInterface interface {
	Initialize(config *TransportConfig) error
	Exec(cmd string, stdin []byte) (*bytes.Buffer, error)
	Put(source string, destination string) error
}
