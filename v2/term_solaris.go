package readline

import "golang.org/x/sys/unix"

type Termios unix.Termios

func getTermios(fd int) (*Termios, error) {
	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, err
	}
	return (*Termios)(termios), nil
}

func setTermios(fd int, termios *Termios) error {
	return unix.IoctlSetTermios(fd, unix.TCSETSF, (*unix.Termios)(termios))
}

// GetSize returns the dimensions of the given terminal.
func GetSize(stdoutFd int) (int, int, error) {
	ws, err := unix.IoctlGetWinsize(stdoutFd, unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}
