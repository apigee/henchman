package henchman

import (
	"errors"
)

type PreprocessError struct {
	Msg   string
	fName string
}

func (pe *preprocessError) Error() string {
	return pe.Msg
}
