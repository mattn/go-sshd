// +build windows

package sshd

import (
	"io"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

type ShellFile struct {
	reader io.ReadCloser
	writer io.WriteCloser
}

func (sf *ShellFile) Read(b []byte) (int, error) {
	return sf.reader.Read(b)
}

func (sf *ShellFile) Write(b []byte) (int, error) {
	return sf.writer.Write(b)
}

func (sf *ShellFile) Close() error {
	sf.reader.Close()
	sf.writer.Close()
	return nil
}

func (sf *ShellFile) Fd() uintptr {
	return 0
}

func startShell(c *exec.Cmd, s ssh.Channel) (*ShellFile, error) {
	// TODO On Windows, shell not echo back.
	writer, err := c.StdinPipe()
	if err != nil {
		return nil, err
	}
	reader, err := c.StdoutPipe()
	if err != nil {
		return nil, err
	}
	err = c.Start()
	if err != nil {
		return nil, err
	}
	return &ShellFile{reader: reader, writer: writer}, nil
}

func setWinsize(fd uintptr, w, h uint32) {
}
