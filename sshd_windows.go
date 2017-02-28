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
	cmd    *exec.Cmd
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

func (s *Server) startShell(c *exec.Cmd, connection ssh.Channel) *ShellFile {
	// TODO On Windows, shell not echo back.
	writer, err := c.StdinPipe()
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		return nil
	}
	reader, err := c.StdoutPipe()
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		return nil
	}
	err = c.Start()
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		return nil
	}
	//pipe session to shell and visa-versa
	go func() {
		io.Copy(connection, reader)
		reader.Close()
	}()
	go func() {
		io.Copy(writer, connection)
		writer.Close()
	}()
	return &ShellFile{reader: reader, writer: writer, cmd: c}
}

func (sf *ShellFile) setWinsize(w, h uint32) {
	// TODO
}
