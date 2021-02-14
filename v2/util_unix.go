// +build darwin dragonfly freebsd netbsd openbsd linux,!appengine solaris

package readline

import (
	"os"
	"syscall"
)

// SuspendMe use to send suspend signal to myself, when we in the raw mode.
// For OSX it need to send to parent's pid
// For Linux it need to send to myself
func SuspendMe() {
	p, _ := os.FindProcess(os.Getppid())
	p.Signal(syscall.SIGTSTP)
	p, _ = os.FindProcess(os.Getpid())
	p.Signal(syscall.SIGTSTP)
}
