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

func ErrNotValidVariable(val interface{}) error {
	return fmt.Errorf("\"%v\" is not a valid variable name", val)
}

func ErrKeyword(val interface{}) error {
	return fmt.Errorf("\"%v\" is a keyword", val)
}

func isKeyword(val string) bool {
	switch val {
	case "vars":
		return true
	case "item":
		return true
	default:
		return false
	}
}
