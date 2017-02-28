// +build windows

package sshd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os/exec"
	"sync"
	"syscall"

	"golang.org/x/crypto/ssh"
)

type ShellFile struct {
	reader io.ReadCloser
	writer io.WriteCloser
	cmd    *exec.Cmd
}

func (s *Server) exec(connection ssh.Channel, cmd string) {
	shell := exec.Command("cmd", "/c", cmd)

	var err error
	var in io.WriteCloser
	var out io.ReadCloser

	// Prepare teardown function
	close := func() {
		in.Close()

		err := shell.Wait()
		var exitStatus int32
		if err != nil {
			if e2, ok := err.(*exec.ExitError); ok {
				if s, ok := e2.Sys().(syscall.WaitStatus); ok {
					exitStatus = int32(s.ExitStatus())
				} else {
					panic(errors.New("Unimplemented for system where exec.ExitError.Sys() is not syscall.WaitStatus."))
				}
			}
		}
		var b bytes.Buffer
		binary.Write(&b, binary.BigEndian, exitStatus)
		connection.SendRequest("exit-status", false, b.Bytes())
		connection.Close()
		s.logger.Printf("Session closed")
	}

	in, err = shell.StdinPipe()
	if err != nil {
		s.logger.Printf("Could not get stdin pipe (%s)", err)
		close()
		return
	}

	out, err = shell.StdoutPipe()
	if err != nil {
		s.logger.Printf("Could not get stdout pipe (%s)", err)
		close()
		return
	}

	err = shell.Start()
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		close()
		return
	}

	//pipe session to shell and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, out)
		once.Do(close)
	}()
	go func() {
		io.Copy(in, connection)
		once.Do(close)
	}()
}

func (s *Server) startShell(connection ssh.Channel) *ShellFile {
	shell := exec.Command(s.shellPath)

	// TODO On Windows, shell not echo back.
	writer, err := shell.StdinPipe()
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		return nil
	}
	reader, err := shell.StdoutPipe()
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		return nil
	}
	err = shell.Start()
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
	return &ShellFile{reader: reader, writer: writer, cmd: shell}
}

func (sf *ShellFile) setWinsize(w, h uint32) {
	// TODO
}
