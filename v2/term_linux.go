package readline

import (
	"syscall"
	"unsafe"
)

type Termios syscall.Termios

// These constants are declared here, rather than importing
// them from the syscall package as some syscall packages, even
// on linux, for example gccgo, do not declare them.
const ioctlReadTermios = 0x5401  // syscall.TCGETS
const ioctlWriteTermios = 0x5402 // syscall.TCSETS

func getTermios(fd int) (*Termios, error) {
	termios := new(Termios)
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlReadTermios, uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if err != 0 {
		return nil, err
	}
	return termios, nil
}

func setTermios(fd int, termios *Termios) error {
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(fd), ioctlWriteTermios, uintptr(unsafe.Pointer(termios)), 0, 0, 0)
	if err != 0 {
		return err
	}
	return nil
}

// GetSize returns the dimensions of the given terminal.
func GetSize(stdoutFd int) (int, int, error) {
	var dimensions [4]uint16
	_, _, err := syscall.Syscall6(syscall.SYS_IOCTL, uintptr(stdoutFd), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&dimensions)), 0, 0, 0)
	if err != 0 {
		return 0, 0, err
	}
	return int(dimensions[1]), int(dimensions[0]), nil
}
