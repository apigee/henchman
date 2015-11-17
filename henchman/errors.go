package henchman

import (
	"fmt"
)

type HenchmanError struct {
	Err    error
	Fields map[string]interface{}
	msg    string
}

func (henchError *HenchmanError) Error() string {
	return henchError.msg
}

func HenchErr(err error, fields map[string]interface{}, extMsg string) error {
	switch val := err.(type) {
	case *HenchmanError:
		if fields != nil {
			MergeLogrusFields(fields, val.Fields, false)
		}
		if extMsg != "" {
			val.msg = (extMsg + " :: " + val.msg)
		}
		return err
	default:
		newFields := fields
		msg := err.Error()

		if newFields == nil {
			newFields = make(map[string]interface{})
		}
		if extMsg != "" {
			msg = (extMsg + " :: " + msg)
		}
		return &HenchmanError{
			Err:    err,
			Fields: newFields,
			msg:    msg,
		}
	}
}

type CustomUnmarshalError struct {
	Err error
}

func (cue *CustomUnmarshalError) Error() string {
	return cue.Err.Error()
}

func ErrWrongType(field interface{}, val interface{}, _type string) error {
	return fmt.Errorf("For field '%v', '%v' is not of type %v", field, val, _type)
}

func ErrNotValidVariable(val interface{}) error {
	return fmt.Errorf("'%v' is not a valid variable name", val)
}

func ErrKeyword(val interface{}) error {
	return fmt.Errorf("'%v' is a keyword", val)
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
