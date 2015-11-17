package henchman

import (
	"fmt"
	log "gopkg.in/Sirupsen/logrus.v0"
	"testing"

	"github.com/stretchr/testify/assert"
	_ "github.com/stretchr/testify/require"
)

func TestHenchErr(t *testing.T) {
	err := fmt.Errorf("Test Error")

	newErr := HenchErr(err, nil, "").(*HenchmanError)
	assert.NotNil(t, newErr.Fields)
	assert.NotNil(t, newErr.Err)
	assert.Equal(t, "Test Error", newErr.Error())

	newErr = HenchErr(newErr, log.Fields{"hello": "world"}, "Message").(*HenchmanError)
	assert.Equal(t, "world", newErr.Fields["hello"])
	assert.Equal(t, "Test Error", newErr.Err.Error())
	assert.Equal(t, "Message :: Test Error", newErr.Error())
}
