package henchman

import (
	"fmt"
	"reflect"
	"sync"
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
			MergeMap(fields, val.Fields, false)
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

func ErrWrongType(field interface{}, val interface{}, _type string) error {
	return fmt.Errorf("For field '%v', '%v' is of typ '%v' not of type %v", field, val, reflect.TypeOf(val), _type)
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
	case "inv":
		return true
	case "current_hostname":
		return true
	default:
		return false
	}
}

func mergeErrs(cs []<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error)

	// Start an output goroutine for each input channel in cs.  output
	// copies values from c to out until c is closed, then calls wg.Done.
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}

	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}

	// Start a goroutine to close out once all the output goroutines are
	// done.  This must start after the wg.Add call.
	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
