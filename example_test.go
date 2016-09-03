package sshd_test

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	sshd "github.com/hnakamur/go-sshd"
	"golang.org/x/crypto/ssh"
)

func ExampleListenAndServe() {
	var (
		address     = flag.String("address", "127.0.0.1:2022", "listen address")
		hostKeyPath = flag.String("host-key", "id_rsa", "the path of the host private key")
		user        = flag.String("user", "foo", "user name")
		password    = flag.String("password", "bar", "user password")
		shell       = flag.String("shell", "bash", "path of shell")
	)
	flag.Parse()

	// In the latest version of crypto/ssh (after Go 1.3), the SSH server type has been removed
	// in favour of an SSH connection type. A ssh.ServerConn is created by passing an existing
	// net.Conn and a ssh.ServerConfig to ssh.NewServerConn, in effect, upgrading the net.Conn
	// into an ssh.ServerConn

	config := &ssh.ServerConfig{
		//Define a function to run when a client attempts a password login
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			// Should use constant-time compare (or better, salt+hash) in a production setting.
			if c.User() == *user && string(pass) == *password {
				return nil, nil
			}
			return nil, fmt.Errorf("password rejected for %q", c.User())
		},
		// You may also explicitly allow anonymous client authentication, though anon bash
		// sessions may not be a wise idea
		// NoClientAuth: true,
	}

	privateBytes, err := ioutil.ReadFile(*hostKeyPath)
	if err != nil {
		log.Fatalf("Failed to load private key (%s); %s", *hostKeyPath, err)
	}

	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatalf("Failed to parse private key; %s", err)
	}

	config.AddHostKey(private)

	server := sshd.NewServer(*shell, config, log.New(os.Stdout, "", 0))
	err = server.ListenAndServe(*address)
	if err != nil {
		log.Fatalf("Failed to listen and serve; %s", err)
	}
}
