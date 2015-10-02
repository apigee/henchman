package henchman

import (
	_ "fmt"
	_ "os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidPasswordAuth(t *testing.T) {
	c := make(TransportConfig)
	c["username"] = "user1"
	c["password"] = "password"
	c["hostname"] = "localhost"
	_, err := NewSSH(&c)

	require.Nil(t, err)
}

func TestInvalidPasswordAuth(t *testing.T) {
	c := make(TransportConfig)
	c["username"] = "user1"
	c["hostname"] = "localhost"
	_, err := NewSSH(&c)
	require.Nil(t, err, "There should have been an error since password isn't present")
}

// func TestSSHExec(t *testing.T) {
// 	c := make(TransportConfig)
// 	c["username"] = "vagrant"
// 	c["keyfile"] = "/Users/sudharsh/.vagrant.d/insecure_private_key"
// 	c["hostname"] = "10.224.192.11"
// 	ssht, err := NewSSH(&c)
// 	buf, err := ssht.Exec("ls -al")
// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}
// 	fmt.Printf("foo - %s\n", buf)
// }

// func TestSCP(t *testing.T) {
// 	c := make(TransportConfig)
// 	c["username"] = "vagrant"
// 	c["keyfile"] = "/Users/sudharsh/.vagrant.d/insecure_private_key"
// 	c["hostname"] = "10.224.192.11"
// 	ssht, err := NewSSH(&c)
// 	err = ssht.Transfer("/tmp/foo.yml", "/home/vagrant")
// 	if err != nil {
// 		t.Errorf(err.Error())
// 	}
// }
