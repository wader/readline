// +build darwin dragonfly freebsd netbsd openbsd linux,!appengine solaris

package readline

import (
	"io"
	"syscall"
)

// State contains the state of a terminal.
type State struct {
	termios Termios
}

// Duplicate duplicates the underlying State.
func (s *State) Duplicate() *State {
	r := *s
	return &r
}

// GetState returns the current state of a terminal which may be useful to
// restore the terminal after a signal.
func GetState(fd uintptr) (*State, error) {
	termios, err := getTermios(fd)
	err = correctErrNo0(err)
	if err != nil {
		return nil, err
	}
	return &State{termios: *termios}, nil
}

// RestoreState restores the terminal connected to the given file descriptor to a
// given state.
func RestoreState(fd uintptr, state *State) error {
	return correctErrNo0(setTermios(fd, &state.termios))
}

// IsTerminal returns true if the given file descriptor is a terminal.
func IsTerminal(fd uintptr) bool {
	_, err := getTermios(fd)
	return err == nil
}

// SetRawMode put the terminal connected to the given file descriptor into raw
// mode and returns the previous state of the terminal so that it can be
// restored.
func SetRawMode(fd uintptr) (*State, error) {
	oldState, err := GetState(fd)
	if err != nil {
		return nil, err
	}

	newState := oldState.Duplicate()
	// This attempts to replicate the behaviour documented for cfmakeraw in
	// the termios(3) manpage.
	newState.termios.Iflag &^= syscall.IGNBRK | syscall.BRKINT | syscall.PARMRK | syscall.ISTRIP | syscall.INLCR | syscall.IGNCR | syscall.ICRNL | syscall.IXON
	// newState.termios.Oflag &^= syscall.OPOST
	newState.termios.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ICANON | syscall.ISIG | syscall.IEXTEN
	newState.termios.Cflag &^= syscall.CSIZE | syscall.PARENB
	newState.termios.Cflag |= syscall.CS8

	newState.termios.Cc[syscall.VMIN] = 1
	newState.termios.Cc[syscall.VTIME] = 0

	return oldState, RestoreState(fd, newState)
}

// SetPasswordMode put the terminal connected to the given file descriptor into password
// mode and returns the previous state of the terminal so that it can be
// restored.
func SetPasswordMode(fd uintptr) (*State, error) {
	oldState, err := GetState(fd)
	if err != nil {
		return nil, err
	}

	newState := oldState.Duplicate()
	newState.termios.Iflag &^= syscall.IGNBRK | syscall.BRKINT
	newState.termios.Iflag |= syscall.ICRNL
	newState.termios.Lflag &^= syscall.ECHO | syscall.ECHONL | syscall.ISIG
	newState.termios.Lflag |= syscall.ICANON // | syscall.ISIG

	return oldState, RestoreState(fd, newState)
}

// ReadPassword reads a line of input from a terminal without local echo.  This
// is commonly used for inputting passwords and other sensitive data. The slice
// returned does not include the \n.
func ReadPassword(fd uintptr) ([]byte, error) {
	oldState, err := SetPasswordMode(fd)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = RestoreState(fd, oldState)
	}()

	var buf [16]byte
	var ret []byte
	for {
		var n int
		n, err = syscall.Read(int(fd), buf[:])
		if err != nil {
			return nil, err
		}
		if n == 0 {
			if len(ret) == 0 {
				return nil, io.EOF
			}
			break
		}
		if buf[n-1] == '\n' {
			n--
		}
		ret = append(ret, buf[:n]...)
		if n < len(buf) {
			break
		}
	}

	return ret, nil
}
