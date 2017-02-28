// +build !windows

package sshd

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/crypto/ssh"

	"github.com/kr/pty"
)

type ShellFile struct {
	file *os.File
}

func (s *Server) exec(connection ssh.Channel, cmd string) {
	shell := exec.Command(s.shellPath, "-c", cmd)

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

// start shell
func (s *Server) startShell(connection ssh.Channel) *ShellFile {
	shell := exec.Command(s.shellPath)

	// Prepare teardown function
	close := func() {
		connection.Close()
		_, err := shell.Process.Wait()
		if err != nil {
			s.logger.Printf("Failed to exit shell (%s)", err)
		}
		s.logger.Printf("Session closed")
	}

	// Allocate a terminal for this channel
	s.logger.Print("Creating pty...")
	file, err := pty.Start(shell)
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		close()
		return nil
	}

	//pipe session to shell and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, file)
		once.Do(close)
	}()
	go func() {
		io.Copy(file, connection)
		once.Do(close)
	}()
	return &ShellFile{file}
}

// SetWinsize sets the size of the given pty.
func (sf *ShellFile) setWinsize(w, h uint32) {
	fd := sf.file.Fd()
	ws := &winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}
