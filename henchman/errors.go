package henchman

import (
	"fmt"
)

type CustomUnmarshalError struct {
	Err error
}

func (cue *CustomUnmarshalError) Error() string {
	return cue.Err.Error()
}

func ErrWrongType(field interface{}, val interface{}, _type string) error {
	return fmt.Errorf("For field \"%v\", \"%v\" is not of type %v", field, val, _type)
}
