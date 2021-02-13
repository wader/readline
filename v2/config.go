package readline

import (
	"io"
)

type Config struct {
	// prompt supports ANSI escape sequence, so we can color some characters
	Prompt string

	// readline will persist historys to file where HistoryFile specified
	HistoryFile string
	// specify the max length of historys, it's 500 by default, set it to -1 to disable history
	HistoryLimit           int
	DisableAutoSaveHistory bool
	// enable case-insensitive history searching
	HistorySearchFold bool

	// AutoCompleter will called once user press TAB
	//AutoComplete AutoCompleter

	InterruptPrompt string
	EOFPrompt       string

	Stdin       io.ReadCloser
	StdinWriter io.Writer
	Stdout      io.Writer
	Stderr      io.Writer

	//?
	EnableMask bool
	MaskRune   rune

	// filter input runes (may be used to disable CtrlZ or for translating some keys to different actions)
	// -> output = new (translated) rune and true/false if continue with processing this one
	FuncFilterInputRune func(rune) (rune, bool)

	ForceUseInteractive bool
}
