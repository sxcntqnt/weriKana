package bankd

import (
	"io"
	"strings"

	"github.com/gliderlabs/ssh"
	"github.com/mattn/go-colorable"
)

// setupPTY returns whether a PTY is allocated and the PTY info.
func setupPTY(s ssh.Session) (bool, ssh.Pty) {
	pty, _, ok := s.Pty() // updated to match current gliderlabs/ssh API
	if !ok {
		return false, ssh.Pty{}
	}
	return true, pty
}

// TerminalWriter wraps a session for PTY-aware output.
type TerminalWriter struct {
	s      ssh.Session
	writer io.Writer
}

// NewTerminalWriter returns a PTY-aware writer.
// ANSI escapes are preserved for PTY sessions, stripped otherwise.
func NewTerminalWriter(s ssh.Session) TerminalWriter {
	hasPTY, _ := setupPTY(s)

	var w io.Writer
	if hasPTY {
		w = s // ANSI allowed
	} else {
		w = colorable.NewNonColorable(s) // ANSI stripped
	}

	return TerminalWriter{
		s:      s,
		writer: w,
	}
}

// Write writes data to the session, stripping ANSI if no PTY.
func (tw TerminalWriter) Write(p []byte) (int, error) {
	hasPTY, _ := setupPTY(tw.s)

	if !hasPTY {
		p = stripANSI(p)
	}

	return tw.writer.Write(p)
}

// stripANSI removes basic ANSI escape sequences (\x1b[...m)
func stripANSI(p []byte) []byte {
	s := string(p)
	for {
		start := strings.Index(s, "\x1b[")
		if start == -1 {
			break
		}
		end := strings.Index(s[start:], "m")
		if end == -1 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	return []byte(s)
}

