package readline

import (
	"errors"
)

var (
	ErrInterrupted = errors.New("interrupted")

	ErrAlreadyInRawMode = errors.New("already in raw mode")
	ErrNotInRawMode     = errors.New("not in raw mode")
)
