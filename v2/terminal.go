package readline

import (
	"io"
	"os"
	"sync"
)

type Terminal struct {
	config      *Config
	stdin       int
	stdout      int
	stderr      int
	stdinReader io.ReadCloser
	stdinWriter io.Writer
	wg          sync.WaitGroup
	mu          sync.Mutex
	oldState    *State
}

func NewTerminal(config *Config) (*Terminal, error) {
	cfg := *config
	if cfg.Stdin == nil {
		cfg.Stdin = os.Stdin
	}
	if cfg.Stdout == nil {
		cfg.Stdout = os.Stdout
	}
	if cfg.Stderr == nil {
		cfg.Stderr = os.Stderr
	}
	t := &Terminal{
		config: &cfg,
		stdin:  int(cfg.Stdin.Fd()),
		stdout: int(cfg.Stdout.Fd()),
		stderr: int(cfg.Stderr.Fd()),
	}
	t.stdinReader, t.stdinWriter = newExtendedStdin(cfg.Stdin)
	t.wg.Add(1)
	go t.ioloop()
	return t, nil
}

func (t *Terminal) Close() error {
	t.wg.Wait()
	_ = t.stdinReader.Close()
	return nil
}

func (t *Terminal) EnterRawMode() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.enterRawMode()
}

func (t *Terminal) enterRawMode() error {
	var err error
	if t.oldState != nil {
		return ErrAlreadyInRawMode
	}
	t.oldState, err = SetRawMode(t.stdin)
	if err != nil {
		return err
	}
	return nil
}

func (t *Terminal) ExitRawMode() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.exitRawMode()
}

func (t *Terminal) exitRawMode() error {
	if t.oldState == nil {
		return ErrNotInRawMode
	}
	return RestoreState(t.stdin, t.oldState)
}

func (t *Terminal) GetSize() (int, int, error) {
	cols, rows, err := GetSize(t.stdout)
	if err != nil {
		cols, rows, err = GetSize(t.stderr)
	}
	return cols, rows, err
}

func (t *Terminal) GetWidth() int {
	w := GetWidth(t.stdout)
	if w < 0 {
		w = GetWidth(t.stderr)
	}
	return w
}

func (t *Terminal) GetHeight() int {
	h := GetHeight(t.stdout)
	if h < 0 {
		h = GetHeight(t.stderr)
	}
	return h
}

func (t *Terminal) ioloop() {
	defer t.wg.Done()

}
