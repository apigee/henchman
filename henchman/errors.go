package henchman

import (
	"fmt"
	log "gopkg.in/Sirupsen/logrus.v0"
)

type HenchmanError struct {
	Err    error
	Fields log.Fields
	msg    string
}

func (he *HenchmanError) Error() string {
	return he.msg
}

func HenchErr(err error, fields log.Fields, extMsg string) error {
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
		var newFields log.Fields = fields
		msg := err.Error()

		if newFields == nil {
			newFields = make(log.Fields)
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
