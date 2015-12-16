package henchman

import (
	"fmt"
	"sync"
)

var printLock sync.Mutex

/**
 * These functions deal with printing
 * They are mainly wrappers that have locks
 */
// recursively print a map.  Only issue is everything is out of order in a map.  Still prints nicely though
func printRecurse(output interface{}, padding string, retVal string) string {
	tmpVal := retVal
	switch output.(type) {
	/*
		case VarsMap:
			for key, val := range output.(VarsMap) {
				switch val.(type) {
				case map[string]interface{}:
					tmpVal += fmt.Sprintf("%s%v:\n", padding, key)
					tmpVal += printRecurse(val, padding+"  ", "")
				default:
					//tmpVal += fmt.Sprintf("%s%v: %v (%v)\n", padding, key, val, reflect.TypeOf(val))
					tmpVal += fmt.Sprintf("%s%v: %v\n", padding, key, val)
				}
			}
	*/
	case map[string]interface{}:
		for key, val := range output.(map[string]interface{}) {
			switch val.(type) {
			case map[string]interface{}:
				tmpVal += fmt.Sprintf("%s%v:\n", padding, key)
				tmpVal += printRecurse(val, padding+"  ", "")
			default:
				//tmpVal += fmt.Sprintf("%s%v: %v (%v)\n", padding, key, val, reflect.TypeOf(val))
				tmpVal += fmt.Sprintf("%s%v: %v\n", padding, key, val)
			}
		}
	default:
		//tmpVal += fmt.Sprintf("%s%v (%s)\n", padding, output, reflect.TypeOf(output))
		tmpVal += fmt.Sprintf("%s%v\n", padding, output)
	}

	return tmpVal
}

// wrapper for Printf and Println with a lock
func Printf(msg string, a ...interface{}) {
	printLock.Lock()
	defer printLock.Unlock()
	fmt.Printf(msg, a...)
}

func Println(msg string) {
	printLock.Lock()
	defer printLock.Unlock()
	fmt.Println(msg)
}

// Does a printf and fills the extra white space
// Just specify the max size to fill to and the string to fill with
func PrintfAndFill(size int, fill string, msg string, a ...interface{}) {
	printLock.Lock()
	defer printLock.Unlock()

	val := fmt.Sprintf(msg, a...)

	var padding string
	for i := 0; i < (size - len(val)); i++ {
		padding += fill
	}
	fmt.Println(val + padding)
}

// Does a Sprintf and fills the extra white space
func SprintfAndFill(size int, fill string, msg string, a ...interface{}) string {
	val := fmt.Sprintf(msg, a...)

	var padding string
	for i := 0; i < (size - len(val)); i++ {
		padding += fill
	}

	return val + padding
}
