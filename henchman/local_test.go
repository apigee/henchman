package henchman

import (
	"path"
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

func TestLocalPut(t *testing.T) {
	fname := "henchman.test.put"
	fpath := writeTempFile([]byte("foobar"), fname)
	defer rmTempFile(path.Join("/tmp", fname))
	defer rmTempFile(path.Join("/tmp", fname+".cp"))
	c := make(TransportConfig)
	local := LocalTransport{}
	err := local.Initialize(&c)
	require.NoError(t, err)
	err = local.Put(fpath, fpath+".cp", "")
	require.NoError(t, err)
}

func TestLocalPutWithSpaces(t *testing.T) {
	fname := "henchman.test put"
	fpath := writeTempFile([]byte("foobar"), fname)
	defer rmTempFile(path.Join("/tmp", fname))
	defer rmTempFile(path.Join("/tmp", fname+".cp"))
	c := make(TransportConfig)
	local := LocalTransport{}
	err := local.Initialize(&c)
	require.NoError(t, err)
	err = local.Put(fpath, fpath+".cp", "")
	require.NoError(t, err)
}
