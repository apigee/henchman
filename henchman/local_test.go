package henchman

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalExec(t *testing.T) {
	c := make(TransportConfig)
	local := LocalTransport{}
	err := local.Initialize(&c)
	require.NoError(t, err)
	buf, err := local.Exec("cat", []byte("foo"), false)

	require.NoError(t, err)
	assert.Equal(t, "foo", strings.TrimSpace(buf.String()), "Expected 'foo'")
}
