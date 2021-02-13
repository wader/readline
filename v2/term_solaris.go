package readline

import "golang.org/x/sys/unix"

type Termios unix.Termios

func getTermios(fd uintptr) (*Termios, error) {
	termios, err := unix.IoctlGetTermios(int(fd), unix.TCGETS)
	if err != nil {
		return nil, err
	}
	return (*Termios)(termios), nil
}

func setTermios(fd uintptr, termios *Termios) error {
	return unix.IoctlSetTermios(int(fd), unix.TCSETSF, (*unix.Termios)(termios))
}

// GetSize returns the dimensions of the given terminal.
func GetSize(fd uintptr) (int, int, error) {
	ws, err := unix.IoctlGetWinsize(int(fd), unix.TIOCGWINSZ)
	if err != nil {
		return 0, 0, err
	}
	return int(ws.Col), int(ws.Row), nil
}
