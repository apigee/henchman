package henchman

import (
	"bytes"
)

type TransportConfig map[string]string

type TransportInterface interface {
	Initialize(config *TransportConfig) error
	Exec(action string) (*bytes.Buffer, error)
}
