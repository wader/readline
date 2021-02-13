// +build darwin dragonfly freebsd netbsd openbsd

package readline

import (
	"syscall"
	"unsafe"
)

type Termios syscall.Termios

func getTermios(fd uintptr) (*Termios, error) {
	termios := new(Termios)
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCGETA, uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if err != 0 {
		return nil, err
	}
	return termios, nil
}

func setTermios(fd uintptr, termios *Termios) error {
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, syscall.TIOCSETA, uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if err != 0 {
		return err
	}
	return nil
}

// GetSize returns the dimensions of the given terminal.
func GetSize(fd uintptr) (int, int, error) {
	var dimensions [4]uint16
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&dimensions)), 0, 0, 0)
	if err != 0 {
		return 0, 0, err
	}
	return int(dimensions[1]), int(dimensions[0]), nil
}
