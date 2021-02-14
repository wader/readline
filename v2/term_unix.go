// +build darwin dragonfly freebsd netbsd openbsd linux,!appengine solaris

package readline

import (
	"io"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
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

// GetState returns the current state of the given file descriptor which may be useful to
// restore the terminal after a signal.
func GetState(fd int) (*State, error) {
	termios, err := getTermios(fd)
	err = correctErrNo0(err)
	if err != nil {
		return nil, err
	}
	return &State{termios: *termios}, nil
}

// RestoreState restores the terminal connected to the given file descriptor to a
// given state.
func RestoreState(fd int, state *State) error {
	return correctErrNo0(setTermios(fd, &state.termios))
}

// SetRawMode put the terminal connected to the given file descriptor into raw
// mode and returns the previous state of the terminal so that it can be
// restored.
func SetRawMode(fd int) (*State, error) {
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
func SetPasswordMode(fd int) (*State, error) {
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

// ReadPassword reads a line of input from a terminal without local echo. This
// is commonly used for inputting passwords and other sensitive data. The slice
// returned does not include the \n.
func ReadPassword(fd int) ([]byte, error) {
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
		n, err = syscall.Read(fd, buf[:])
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

// IsTerminal returns true if the given file descriptor is a terminal.
func IsTerminal(fd int) bool {
	_, err := getTermios(fd)
	return err == nil
}

// IsScreenTerminal returns true if the current screen is a terminal.
func IsScreenTerminal() bool {
	return IsTerminal(syscall.Stdin) && (IsTerminal(syscall.Stdout) || IsTerminal(syscall.Stderr))
}

// GetWidth gets width of the given file descriptor. If error occurs, it returns -1.
func GetWidth(stdoutFd int) int {
	cols, _, err := GetSize(stdoutFd)
	if err != nil {
		return -1
	}
	return cols
}

// GetScreenWidth gets width of the current screen. If error occurs, it returns -1.
func GetScreenWidth() int {
	w := GetWidth(syscall.Stdout)
	if w < 0 {
		w = GetWidth(syscall.Stderr)
	}
	return w
}

// GetHeight gets height of the given file descriptor. If error occurs, it returns -1.
func GetHeight(stdoutFd int) int {
	_, rows, err := GetSize(stdoutFd)
	if err != nil {
		return -1
	}
	return rows
}

// GetScreenHeight gets height of the current screen. If error occurs, it returns -1.
func GetScreenHeight() int {
	h := GetHeight(syscall.Stdout)
	if h < 0 {
		h = GetHeight(syscall.Stderr)
	}
	return h
}

var (
	onScreenWidthChangedOnce     sync.Once
	onScreenWidthChangedCallback *func()
)

// OnScreenWidthChanged callbacks to func when the current screen with is changed.
func OnScreenWidthChanged(callback func()) (oldCallback func()) {
	oldCallbackPtr := atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(&onScreenWidthChangedCallback)), unsafe.Pointer(&callback))
	if oldCallbackPtr != nil {
		oldCallback = *(*func())(oldCallbackPtr)
	}
	onScreenWidthChangedOnce.Do(func() {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				if onScreenWidthChangedCallback != nil && *onScreenWidthChangedCallback != nil {
					(*onScreenWidthChangedCallback)()
				}
			}
		}()
	})
	return oldCallback
}
