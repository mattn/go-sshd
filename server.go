// Package sshd provides a subset of the ssh server protocol.
// Only supported request types are shell, pty-req, and window-change.
package sshd

import (
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"github.com/kr/pty"
	"golang.org/x/crypto/ssh"
)

// Server is the sshd server.
type Server struct {
	shellPath string
	config    *ssh.ServerConfig
	logger    *log.Logger
	listener  net.Listener

	mu     sync.Mutex
	closed bool
}

// NewServer creates a sshd server.
// The shellPath is the path of the shell (e.g., "bash").
// You can pass nil as logger if you want to disable log outputs.
func NewServer(shellPath string, config *ssh.ServerConfig, logger *log.Logger) *Server {
	if logger == nil {
		logger = log.New(ioutil.Discard, "", 0)
	}
	return &Server{shellPath: shellPath, config: config, logger: logger}
}

// ListenAndServe let the server listen and serve.
func (s *Server) ListenAndServe(addr string) error {
	// Once a ServerConfig has been configured, connections can be accepted.
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("Failed to listen on %s (%s)", addr, err)
	}
	return s.Serve(listener)
}

// Serve let the server accept incoming connections and handle them.
func (s *Server) Serve(l net.Listener) error {
	s.listener = l
	for {
		tcpConn, err := l.Accept()
		if err != nil {
			if s.isClosed() {
				return nil
			}
			return fmt.Errorf("Failed to accept incoming connection (%s)", err)
		}
		// Before use, a handshake must be performed on the incoming net.Conn.
		_, chans, reqs, err := ssh.NewServerConn(tcpConn, s.config)
		if err != nil {
			return err
		}

		// Discard all global out-of-band Requests
		go ssh.DiscardRequests(reqs)
		// Accept all channels
		go s.handleChannels(chans)
	}
}

// Close stops the server.
func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	s.mu.Unlock()
	return s.listener.Close()
}

func (s *Server) isClosed() bool {
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	return closed
}

func (s *Server) handleChannels(chans <-chan ssh.NewChannel) {
	// Service the incoming Channel channel in go routine
	for newChannel := range chans {
		go s.handleChannel(newChannel)
	}
}

func (s *Server) handleChannel(newChannel ssh.NewChannel) {
	// Since we're handling a shell, we expect a
	// channel type of "session". The also describes
	// "x11", "direct-tcpip" and "forwarded-tcpip"
	// channel types.
	if t := newChannel.ChannelType(); t != "session" {
		newChannel.Reject(ssh.UnknownChannelType, fmt.Sprintf("unknown channel type: %s", t))
		return
	}

	// At this point, we have the opportunity to reject the client's
	// request for another logical connection
	connection, requests, err := newChannel.Accept()
	if err != nil {
		s.logger.Printf("Could not accept channel (%s)", err)
		return
	}

	// Fire up bash for this session
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
	shellf, err := pty.Start(shell)
	if err != nil {
		s.logger.Printf("Could not start pty (%s)", err)
		close()
		return
	}

	//pipe session to shell and visa-versa
	var once sync.Once
	go func() {
		io.Copy(connection, shellf)
		once.Do(close)
	}()
	go func() {
		io.Copy(shellf, connection)
		once.Do(close)
	}()

	// Sessions have out-of-band requests such as "shell", "pty-req" and "env"
	go func() {
		for req := range requests {
			switch req.Type {
			case "shell":
				// We only accept the default shell
				// (i.e. no command in the Payload)
				if len(req.Payload) == 0 {
					req.Reply(true, nil)
				}
			case "pty-req":
				termLen := req.Payload[3]
				w, h := parseDims(req.Payload[termLen+4:])
				setWinsize(shellf.Fd(), w, h)
				// Responding true (OK) here will let the client
				// know we have a pty ready for input
				req.Reply(true, nil)
			case "window-change":
				w, h := parseDims(req.Payload)
				setWinsize(shellf.Fd(), w, h)
			}
		}
	}()
}

// =======================

// parseDims extracts terminal dimensions (width x height) from the provided buffer.
func parseDims(b []byte) (uint32, uint32) {
	w := binary.BigEndian.Uint32(b)
	h := binary.BigEndian.Uint32(b[4:])
	return w, h
}

// ======================

// Winsize stores the Height and Width of a terminal.
type winsize struct {
	Height uint16
	Width  uint16
	x      uint16 // unused
	y      uint16 // unused
}

// SetWinsize sets the size of the given pty.
func setWinsize(fd uintptr, w, h uint32) {
	ws := &winsize{Width: uint16(w), Height: uint16(h)}
	syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCSWINSZ), uintptr(unsafe.Pointer(ws)))
}

// Borrowed from https://github.com/creack/termios/blob/master/win/win.go
