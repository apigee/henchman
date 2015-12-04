package henchman

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	_ "path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	ECHO          = 53
	TTY_OP_ISPEED = 128
	TTY_OP_OSPEED = 129
)

func loadPEM(file string) (ssh.Signer, error) {
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	key, err := ssh.ParsePrivateKey(buf)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func ClientKeyAuth(keyFile string) (ssh.AuthMethod, error) {
	key, err := loadPEM(keyFile)
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"key_file": keyFile,
		}, "")
	}
	return ssh.PublicKeys(key), err
}

func PasswordAuth(pass string) (ssh.AuthMethod, error) {
	return ssh.Password(pass), nil
}

type SSHTransport struct {
	Host   string
	Port   uint16
	Config *ssh.ClientConfig
}

func (sshTransport *SSHTransport) Initialize(config *TransportConfig) error {
	_config := *config

	// Get hostname and port
	sshTransport.Host = _config["hostname"]
	port, parseErr := strconv.ParseUint(_config["port"], 10, 16)
	if parseErr != nil || port == 0 {
		/*
			if Debug {
				log.Debug("Assuming default port to be 22")
			}
		*/
		sshTransport.Port = 22
	} else {
		sshTransport.Port = uint16(port)
	}
	if sshTransport.Host == "" {
		return HenchErr(fmt.Errorf("Need a hostname"), nil, "SSH transport")
	}
	username := _config["username"]
	if username == "" {
		return HenchErr(fmt.Errorf("Need a username"), nil, "SSH transport")
	}
	var auth ssh.AuthMethod
	var authErr error

	password, present := _config["password"]
	if password == "" || !present {
		keyfile, present := _config["keyfile"]
		if !present {
			return HenchErr(fmt.Errorf("Invalid SSH Keyfile"), nil, "SSH transport")
		}
		auth, authErr = ClientKeyAuth(keyfile)
	} else {
		auth, authErr = PasswordAuth(password)
	}

	if authErr != nil {
		return HenchErr(authErr, nil, "SSH transport auth error")
	}
	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{auth},
	}
	sshTransport.Config = sshConfig
	return nil
}

func (sshTransport *SSHTransport) getClientSession() (*ssh.Client, *ssh.Session, error) {
	address := fmt.Sprintf("%s:%d", sshTransport.Host, sshTransport.Port)
	client, err := ssh.Dial("tcp", address, sshTransport.Config)
	if err != nil {
		return nil, nil, HenchErr(err, nil, "")
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, nil, HenchErr(err, nil, "")
	}
	return client, session, nil

}

/*
func (sshTransport *SSHTransport) execCmd(session *ssh.Session, cmd string) (*bytes.Buffer, error) {
	var b bytes.Buffer
	modes := ssh.TerminalModes{
		ECHO:          0,
		TTY_OP_ISPEED: 14400,
		TTY_OP_OSPEED: 1,
	}
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		return nil, HenchErr(err, nil, "request for psuedo terminal failed")
	}
	session.Stdout = &b
	if err := session.Run(cmd); err != nil {
		return nil, HenchErr(fmt.Errorf(b.String()), nil, "")
	}
	return &b, nil
}
*/

func (sshTransport *SSHTransport) Exec(cmd string, stdin []byte, sudoEnabled bool) (*bytes.Buffer, error) {
	client, session, err := sshTransport.getClientSession()
	if err != nil {
		return nil, HenchErr(err, map[string]interface{}{
			"host": sshTransport.Host,
		}, fmt.Sprintf("Couldn't dial into %s", sshTransport.Host))
	}

	defer client.Close()
	defer session.Close()
	if sudoEnabled {
		cmd = fmt.Sprintf("/bin/bash -c 'sudo -H -u root %s'", cmd)
	}

	cmd = fmt.Sprintf("echo '%s' | %s", stdin, cmd)
	/*
		if Debug {
			log.Debug(cmd)
		}
	*/
	//bytesBuf, err := sshTransport.execCmd(session, cmd)
	bytesSlice, err := session.CombinedOutput(cmd)
	if err != nil {
		return nil, HenchErr(err, nil, "While executing command")
	}

	return bytes.NewBuffer(bytesSlice), nil
}

// source is the source of the file/folder
// destination is the path of the FOLDER where source will be transferred to
// NOTE: source will keep it original name when transferred
func (sshTransport *SSHTransport) Put(source, destination string, srcType string) error {
	client, session, err := sshTransport.getClientSession()
	if err != nil {
		return HenchErr(err, map[string]interface{}{
			"host": sshTransport.Host,
		}, fmt.Sprintf("Couldn't dial into %s", sshTransport.Host))
	}
	defer client.Close()
	defer session.Close()

	sftp, err := sftp.NewClient(client)
	if err != nil {
		return HenchErr(err, map[string]interface{}{
			"host": sshTransport.Host,
		}, "Error creating sftp client")
	}
	defer sftp.Close()

	dstPieces := strings.Split(destination, "/")

	// it will always equal home
	if dstPieces[0] == "${HOME}" {
		dstPieces[0] = "."
	}

	newDst := strings.Join(dstPieces, "/")

	// if the src is a dir
	// tar the dir, copy it over, untar it
	// remove tar files on both ends
	if srcType == "dir" {
		// This is done so hosts won't remove tars used by other file
		tarredFile := sshTransport.Host + "_" + filepath.Base(source) + ".tar"

		// create the local tar
		err := tarit(source, tarredFile, nil)

		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"dir": source,
			}, "Failed to tar dir to transport")
		}
		defer os.Remove(tarredFile)

		sourceBuf, err := ioutil.ReadFile(tarredFile)
		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"dir":  source,
				"file": tarredFile,
			}, "Failed to read tar")
		}

		tarredFilePath := filepath.Join(newDst, tarredFile)
		f, err := sftp.Create(tarredFilePath)
		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"host": sshTransport.Host,
				"file": newDst,
			}, "Failed to create remote file")
		}

		if _, err := f.Write(sourceBuf); err != nil {
			return HenchErr(err, map[string]interface{}{
				"host":   sshTransport.Host,
				"file":   destination,
				"source": source,
			}, "Error writing to remote file ")
		}

		cmd := fmt.Sprintf("tar -xvf %s -C %s && rm -rf %s", tarredFilePath, newDst, tarredFilePath)
		_, err = session.CombinedOutput(cmd)
		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"host": sshTransport.Host,
				"file": tarredFilePath,
			}, "While untarring on remote")
		}
	} else {
		f, err := sftp.Create(filepath.Join(newDst, filepath.Base(source)))
		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"host": sshTransport.Host,
				"file": destination,
			}, "Failed to create remote file")
		}

		sourceBuf, err := ioutil.ReadFile(source)
		if err != nil {
			return HenchErr(err, map[string]interface{}{
				"file": source,
			}, "")
		}

		if _, err := f.Write(sourceBuf); err != nil {
			return HenchErr(err, map[string]interface{}{
				"host":   sshTransport.Host,
				"file":   destination,
				"source": source,
			}, "Error writing to remote file ")
		}
	}

	return nil
}

func NewSSH(config *TransportConfig) (*SSHTransport, error) {
	ssht := SSHTransport{}
	return &ssht, ssht.Initialize(config)
}
