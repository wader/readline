package readline

import (
	"bufio"
	"context"
	"io"
	"os"
	"sync"
	"unicode/utf8"

	"github.com/goinsane/readline/v2/runeutil"
)

type Terminal struct {
	config              *Config
	stdin               int
	stdout              int
	stderr              int
	screenBrokenPipeCh  chan struct{}
	screenSizeChangedCh chan struct{}
	lineResultCh        chan lineResult
	rb                  *runeutil.RuneBuffer
	stdinReader         io.ReadCloser
	stdinWriter         io.Writer
	ctx                 context.Context
	ctxCancel           context.CancelFunc
	wg                  sync.WaitGroup
	onceClose           sync.Once
	mu                  sync.Mutex
	oldState            *State
	ioErr               error
}

func NewTerminal(config Config) (*Terminal, error) {
	if config.Stdin == nil {
		config.Stdin = os.Stdin
	}
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	t := &Terminal{
		config:              &config,
		stdin:               int(config.Stdin.Fd()),
		stdout:              int(config.Stdout.Fd()),
		stderr:              int(config.Stderr.Fd()),
		screenBrokenPipeCh:  make(chan struct{}, 1),
		screenSizeChangedCh: make(chan struct{}, 1),
		lineResultCh:        make(chan lineResult, 1),
	}
	t.rb = runeutil.NewRuneBuffer(config.Stdout, config.Prompt, config.Mask, config.ForceUseInteractive, GetWidth(t.stdout))
	t.stdinReader, t.stdinWriter = newExtendedStdin(config.Stdin)
	t.ctx, t.ctxCancel = context.WithCancel(context.Background())
	RegisterOnScreenBrokenPipe(t.screenBrokenPipeCh)
	RegisterOnScreenSizeChanged(t.screenSizeChangedCh)
	t.wg.Add(1)
	go t.ioloop()
	return t, nil
}

func (t *Terminal) Close() error {
	var err error
	t.onceClose.Do(func() {
		t.ctxCancel()
		_ = t.stdinReader.Close()
		t.wg.Wait()
		err = t.ExitRawMode()
	})
	return err
}

func (t *Terminal) Stdin() *os.File {
	return t.config.Stdin
}

func (t *Terminal) Stdout() *os.File {
	return t.config.Stdout
}

func (t *Terminal) Stderr() *os.File {
	return t.config.Stderr
}

func (t *Terminal) StdinWriter() io.Writer {
	return t.stdinWriter
}

func (t *Terminal) Write(b []byte) (int, error) {
	return t.config.Stdout.Write(b)
}

// WriteStdin prefill the next Stdin fetch
// Next time you call ReadLine() this value will be writen before the user input
func (t *Terminal) WriteStdin(b []byte) (int, error) {
	return t.stdinWriter.Write(b)
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
	if err := RestoreState(t.stdin, t.oldState); err != nil {
		return err
	}
	t.oldState = nil
	return nil
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

func (t *Terminal) ReadSlice() ([]byte, error) {
	return t.ReadSliceContext(context.Background())
}

func (t *Terminal) ReadSliceContext(ctx context.Context) (line []byte, err error) {
	t.mu.Lock()
	err = t.ioErr
	t.mu.Unlock()
	if err != nil {
		return nil, err
	}
	err = t.EnterRawMode()
	if err != nil {
		return nil, err
	}
	defer func() {
		e := t.ExitRawMode()
		if err == nil {
			err = e
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c := <-t.lineResultCh:
		return c.Line, c.Err
	}
}

func (t *Terminal) ioloop() {
	defer t.wg.Done()

	sendLineResult := func(line []byte, err error) {
		r := lineResult{
			Line: make([]byte, len(line)),
			Err:  err,
		}
		copy(r.Line, line)
		select {
		case t.lineResultCh <- r:
		default:
		}
	}

	lineBuf := make([]byte, 0, 1024)
	appendLineBuf := func(p []byte) {
		bufSize := len(lineBuf)
		newBufSize := bufSize + len(p)
		if cap(lineBuf) < newBufSize {
			newBuf := make([]byte, bufSize, 2*newBufSize)
			copy(newBuf, lineBuf)
			lineBuf = newBuf
		}
		lineBuf = append(lineBuf, p...)
	}

	escaped := false
	escBuf := make([]byte, 0, 16)

	br := bufio.NewReader(t.stdinReader)
	var err error
	for t.ctx.Err() == nil && err == nil {
		var b byte
		var p []byte
		b, err = br.ReadByte()
		if err != nil {
			break
		}
		if b >= utf8.RuneSelf && !escaped {
			_ = br.UnreadByte()
			var r rune
			r, _, err = br.ReadRune()
			if err != nil {
				break
			}
			var utf8Array [utf8.UTFMax]byte
			p = utf8Array[:utf8.EncodeRune(utf8Array[:], r)]
		} else {
			p = []byte{b}
		}

		if b == CharEscape || escaped {
			if !escaped {
				escaped = true
				escBuf = escBuf[:0]
			} else {
				escBuf = append(escBuf, p...)
			}
			escKeyPair := decodeEscapeKeyPair(escBuf)
			if escKeyPair != nil {
				t.escape(escKeyPair)
				escaped = false
				p = escKeyPair.Remainder
			} else {
				if len(escBuf) < cap(escBuf) {
					continue
				}
				escaped = false
				p = append([]byte{CharEscape}, escBuf...)
			}
		}

		if len(p) <= 0 {
			continue
		}

		switch p[0] {
		case CharLineStart:

		case CharBackward:

		case CharInterrupt:

		case CharDelete:
			err = io.EOF

		case CharLineEnd:

		case CharForward:

		case CharBell:

		case CharCtrlH, CharBackspace:

		case CharTab:

		case CharCtrlJ, CharEnter:
			t.print('\n')
			sendLineResult(lineBuf, err)

		case CharKill:

		case CharClear:

		case CharNext:

		case CharPrev:

		case CharBckSearch:

		case CharFwdSearch:

		case CharTranspose:

		default:
			c := encodeControlChars(p)
			r, _ := utf8.DecodeRune(c)
			appendLineBuf(c)
			t.rb.WriteRune(r)

		}
	}

	t.mu.Lock()
	t.ioErr = err
	t.mu.Unlock()

	sendLineResult(lineBuf, err)
}

func (t *Terminal) escape(escKeyPair *escapeKeyPair) {

}

func (t *Terminal) print(p ...byte) {
	_, _ = t.Write(p)
}

func (t *Terminal) opLineStart() {

}

func (t *Terminal) opBackward() {

}

func (t *Terminal) opLineEnd() {

}

func (t *Terminal) opForward() {

}

func (t *Terminal) opBackSpace() {

}

func (t *Terminal) opTab() {

}

func (t *Terminal) opKill() {

}

func (t *Terminal) opClear() {

}

func (t *Terminal) opNext() {

}

func (t *Terminal) opPrev() {

}

func (t *Terminal) opBckSearch() {

}

func (t *Terminal) opFwdSearch() {

}

func (t *Terminal) opTranspose() {

}
