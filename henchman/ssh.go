package henchman

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strconv"

	"code.google.com/p/go.crypto/ssh"
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
		log.Printf("Assuming default port to be 22\n")
		sshTransport.Port = 22
	} else {
		sshTransport.Port = uint16(port)
	}
	if sshTransport.Host == "" {
		return errors.New("Need a hostname")
	}
	username := _config["username"]
	if username == "" {
		return errors.New("Need a username")
	}
	var auth ssh.AuthMethod
	var authErr error

	password, present := _config["password"]
	if password == "" || !present {
		keyfile, present := _config["keyfile"]
		if !present {
			return errors.New("Invalid SSH Keyfile")
		}
		auth, authErr = ClientKeyAuth(keyfile)
	} else {
		auth, authErr = PasswordAuth(password)
	}

	if authErr != nil {
		return authErr
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
		return nil, nil, err
	}
	session, err := client.NewSession()
	if err != nil {
		return nil, nil, err
	}
	return client, session, nil

}

func (sshTransport *SSHTransport) execCmd(session *ssh.Session, cmd string) (*bytes.Buffer, error) {
	var b bytes.Buffer
	modes := ssh.TerminalModes{
		ECHO:          0,
		TTY_OP_ISPEED: 14400,
		TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: " + err.Error())
	}
	session.Stdout = &b
	if err := session.Run(cmd); err != nil {
		return nil, errors.New(b.String())
	}
	return &b, nil
}

func (sshTransport *SSHTransport) Exec(cmd string, stdin []byte, sudoEnabled bool) (*bytes.Buffer, error) {
	client, session, err := sshTransport.getClientSession()
	if err != nil {
		log.Printf("Couldn't dial in to %s", sshTransport.Host)
		return nil, err
	}
	defer client.Close()
	defer session.Close()
	if sudoEnabled {
		cmd = fmt.Sprintf("/bin/bash -c 'sudo -H -u root %s'", cmd)
	}
	cmd = fmt.Sprintf("echo '%s' | %s", stdin, cmd)
	log.Println(cmd)
	return sshTransport.execCmd(session, cmd)
}

func (sshTransport *SSHTransport) Put(source, destination string, dstType string) error {
	client, session, err := sshTransport.getClientSession()
	if err != nil {
		log.Printf("Couldn't dial in to %s", sshTransport.Host)
		return err
	}
	defer client.Close()
	defer session.Close()
	sourceBuf, err := ioutil.ReadFile(source)
	if err != nil {
		log.Printf("Error reading file - %s: %s\n", source, err.Error())
		return err
	}
	_, sourcePath := path.Split(source)
	go func() {
		pipe, err := session.StdinPipe()
		if err != nil {
			log.Printf("Error opening pipe - %s\n", err.Error())
			return
		}
		defer pipe.Close()
		buf := string(sourceBuf)
		if dstType == "dir" {
			fmt.Fprintln(pipe, "C0700", len(buf), sourcePath)
		} else {
			fmt.Fprintln(pipe, "C0644", len(buf), sourcePath)
		}
		fmt.Fprint(pipe, buf)
		fmt.Fprint(pipe, "\x00")
	}()
	//default directory scp command
	remoteCommand := fmt.Sprintf("mkdir -p %s && cd %s && /usr/bin/scp -qrt ./", destination, destination)
	if dstType == "file" {
		remoteCommand = fmt.Sprintf("/usr/bin/scp -t %s", destination)
	}
	if err := session.Run(remoteCommand); err != nil {
		log.Printf("Error doing scp - %s\n", err.Error())
		return err
	}
	return nil
}

func NewSSH(config *TransportConfig) (*SSHTransport, error) {
	ssht := SSHTransport{}
	return &ssht, ssht.Initialize(config)
}
