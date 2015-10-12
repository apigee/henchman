package henchman

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetHenchmanVars(t *testing.T) {
	vars := VarsMap{
		"ulimit":        100,
		"henchman_user": "hello",
		"henchman_pass": "hi",
	}
	henchmanVars := GetHenchmanVars(vars)
	assert.Equal(t, 2, len(henchmanVars), "Length of henchmanVars was expected to be 2")
	assert.Equal(t, "hello", henchmanVars["user"], "Length of henchmanVars was expected to be 2")
	assert.Equal(t, "hi", henchmanVars["pass"], "Length of henchmanVars was expected to be 2")
}
