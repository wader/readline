package readline

import (
	"errors"
)

var (
	ErrAlreadyInRawMode = errors.New("already in raw mode")
	ErrNotInRawMode     = errors.New("not in raw mode")
)
