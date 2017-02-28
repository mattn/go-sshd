// +build !windows

package sshd

import (
	"io"
	"os/exec"
	"syscall"
	"unsafe"

	"github.com/kr/pty"
)

type ShellFile File

// start shell
func startShell(c *exec.Cmd) (io.ReadWriteCloser, error) {
	return pty.Start(c)
}

// SetWinsize sets the size of the given pty.
func setWinsize(fd uintptr, w, h uint32) {
	ws := &winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}
