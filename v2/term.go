package readline

import (
	"sync"
	"syscall"
)

func init() {
	go loopOnScreenBrokenPipe()
	go loopOnScreenSizeChanged()
}

// IsScreenTerminal returns true if the current screen is a terminal.
func IsScreenTerminal() bool {
	return IsStdinTerminal() && (IsStdoutTerminal() || IsStderrTerminal())
}

// IsStdinTerminal returns true if stdin is a terminal.
func IsStdinTerminal() bool {
	return IsTerminal(syscall.Stdin)
}

// IsStdoutTerminal returns true if stdout is a terminal.
func IsStdoutTerminal() bool {
	return IsTerminal(syscall.Stdout)
}

// IsStderrTerminal returns true if stderr is a terminal.
func IsStderrTerminal() bool {
	return IsTerminal(syscall.Stderr)
}

// GetScreenSize gets size of the current screen.
func GetScreenSize() (int, int, error) {
	cols, rows, err := GetSize(syscall.Stdout)
	if err != nil {
		cols, rows, err = GetSize(syscall.Stderr)
	}
	return cols, rows, err
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
	onScreenBrokenPipeMu  sync.Mutex
	onScreenBrokenPipeMap = make(map[chan<- struct{}]struct{}, 16)
)

func loopOnScreenBrokenPipe() {
	for range screenBrokenPipeCh {
		onScreenBrokenPipeMu.Lock()
		for ch := range onScreenBrokenPipeMap {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		onScreenBrokenPipeMu.Unlock()
	}
}

func RegisterOnScreenBrokenPipe(ch chan<- struct{}) {
	onScreenBrokenPipeMu.Lock()
	defer onScreenBrokenPipeMu.Unlock()
	if _, ok := onScreenBrokenPipeMap[ch]; !ok {
		onScreenBrokenPipeMap[ch] = struct{}{}
	}
}

func UnregisterOnScreenBrokenPipe(ch chan<- struct{}) {
	onScreenBrokenPipeMu.Lock()
	defer onScreenBrokenPipeMu.Unlock()
	if _, ok := onScreenBrokenPipeMap[ch]; ok {
		delete(onScreenBrokenPipeMap, ch)
	}
}

var (
	onScreenSizeChangedMu  sync.Mutex
	onScreenSizeChangedMap = make(map[chan<- struct{}]struct{}, 16)
)

func loopOnScreenSizeChanged() {
	for range screenSizeChangedCh {
		onScreenSizeChangedMu.Lock()
		for ch := range onScreenSizeChangedMap {
			select {
			case ch <- struct{}{}:
			default:
			}
		}
		onScreenSizeChangedMu.Unlock()
	}
}

func RegisterOnScreenSizeChanged(ch chan<- struct{}) {
	onScreenSizeChangedMu.Lock()
	defer onScreenSizeChangedMu.Unlock()
	if _, ok := onScreenSizeChangedMap[ch]; !ok {
		onScreenSizeChangedMap[ch] = struct{}{}
	}
}

func UnregisterOnScreenSizeChanged(ch chan<- struct{}) {
	onScreenSizeChangedMu.Lock()
	defer onScreenSizeChangedMu.Unlock()
	if _, ok := onScreenSizeChangedMap[ch]; ok {
		delete(onScreenSizeChangedMap, ch)
	}
}
