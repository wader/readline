package readline

import (
	"bufio"
	"context"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"unicode/utf8"

	"github.com/goinsane/readline/v2/runeutil"

	"github.com/goinsane/xcontext"
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
	lckr                xcontext.Locker
	oldState            *State
	ioErr               atomic.Value
}

func NewTerminal(config Config) (*Terminal, error) {
	var err error
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
	interactive := IsTerminal(t.stdin)
	if config.ForceUseInteractive {
		interactive = true
	}
	t.rb, err = runeutil.NewRuneBuffer(config.Stdout, config.Prompt, config.Mask, interactive, t.GetWidth())
	if err != nil {
		return nil, err
	}
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
		UnregisterOnScreenBrokenPipe(t.screenBrokenPipeCh)
		UnregisterOnScreenSizeChanged(t.screenSizeChangedCh)
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
	t.lckr.Lock()
	defer t.lckr.Unlock()
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
	t.lckr.Lock()
	defer t.lckr.Unlock()
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

func (t *Terminal) ReadBytes() ([]byte, error) {
	return t.ReadBytesContext(context.Background())
}

func (t *Terminal) ReadBytesContext(ctx context.Context) (line []byte, err error) {
	err = t.lckr.LockContext(ctx)
	if err != nil {
		return nil, err
	}
	defer t.lckr.Unlock()
	ioErr := t.ioErr.Load()
	if ioErr != nil {
		return nil, ioErr.(error)
	}
	err = t.enterRawMode()
	if err != nil {
		return nil, err
	}
	defer t.exitRawMode()
	t.rb.Refresh(nil)
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case c := <-t.lineResultCh:
		return c.Line, c.Err
	}
}

func (t *Terminal) ReadString() (string, error) {
	return t.ReadStringContext(context.Background())
}

func (t *Terminal) ReadStringContext(ctx context.Context) (string, error) {
	p, err := t.ReadBytesContext(ctx)
	return string(p), err
}

func (t *Terminal) ReadLine() (string, error) {
	return t.ReadString()
}

func (t *Terminal) ReadLineContext(ctx context.Context) (string, error) {
	return t.ReadLineContext(ctx)
}

func (t *Terminal) ioloop() {
	defer t.wg.Done()

	br := bufio.NewReader(t.stdinReader)
	escaped := false
	escBuf := make([]byte, 0, 16)

	var err error
	for err == nil {
		err = t.ctx.Err()
		if err != nil {
			continue
		}
		var b byte
		var p []byte
		b, err = br.ReadByte()
		if err != nil {
			if isInterruptedSyscall(err) {
				err = nil
				escaped = false
				escBuf = escBuf[:0]
			}
			continue
		}
		if b >= utf8.RuneSelf && !escaped {
			_ = br.UnreadByte()
			var r rune
			r, _, err = br.ReadRune()
			if err != nil {
				continue
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
			if escKeyPair != nil && t.escape(escKeyPair) {
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
			t.opLineStart()

		case CharBackward:
			t.opBackward()

		case CharInterrupt:
			err = ErrInterrupted

		case CharDelete:
			err = io.EOF

		case CharLineEnd:
			t.opLineEnd()

		case CharForward:
			t.opForward()

		case CharBell:
			t.opBell()

		case CharCtrlH, CharBackspace:
			t.opBackspace()

		case CharTab:
			t.opTab()

		case CharCtrlJ, CharEnter:
			t.opEnter()

		case CharKill:
			t.opKill()

		case CharClear:
			t.opClear()

		case CharNext:
			t.opNext()

		case CharPrev:
			t.opPrev()

		case CharBckSearch:
			t.opBckSearch()

		case CharFwdSearch:
			t.opFwdSearch()

		case CharTranspose:
			t.opTranspose()

		case CharKillFront:
			t.opKillFront()

		case CharYank:
			t.opYank()

		default:
			t.rb.WriteBytes(encodeControlChars(p))

		}
	}

	if xcontext.IsContextError(err) {
		err = io.EOF
	}
	t.ioErr.Store(err)
	t.sendLineResult(t.rb.Bytes(), err)
}

func (t *Terminal) escape(escKeyPair *escapeKeyPair) bool {
	switch escKeyPair.Char {
	case CharTranspose:
		t.opTranspose()

	case CharEscape:

	case CharBackspace:
		t.opBackEscapeWord()

	case 'O', '[':
		return t.escapeEx(escKeyPair)

	case 'b':
		t.opBackward()

	case 'd':
		t.opDelete()

	case 'f':
		t.opForward()

	default:

	}

	return true
}

func (t *Terminal) escapeEx(escKeyPair *escapeKeyPair) bool {
	switch escKeyPair.Type {
	case '\x00':
		return false

	case '~':
		t.escapeTilda(escKeyPair)
		return true

	case 'R':
		t.escapeR(escKeyPair)
		return true

	default:
		if escKeyPair.Attribute <= 0 && escKeyPair.Attribute2 < 0 {
			switch escKeyPair.Type {
			case 'A':
				t.opPrev()

			case 'B':
				t.opNext()

			case 'C':
				t.opForward()

			case 'D':
				t.opBackward()

			case 'E':
				//

			case 'F':
				t.opLineEnd()

			case 'H':
				t.opLineStart()

			}
		}

	}

	return true
}

func (t *Terminal) escapeTilda(escKeyPair *escapeKeyPair) {
	if escKeyPair.Attribute2 < 0 {
		switch escKeyPair.Attribute {
		case 1:
			t.opLineStart()

		case 2:
			// insert mode

		case 3:
			t.opDelete()

		case 4:
			t.opLineEnd()

		case 5:
			// pageup

		case 6:
			// pagedown

		case 7:
			t.opLineStart()

		case 8:
			t.opLineEnd()

		}
	}
}

func (t *Terminal) escapeR(escKeyPair *escapeKeyPair) {
	if escKeyPair.Attribute >= 0 && escKeyPair.Attribute2 >= 0 {
		t.screenSizeChanged(escKeyPair.Attribute2, escKeyPair.Attribute)
	}
}

func (t *Terminal) sendLineResult(line []byte, e error) {
	r := lineResult{
		Line: line,
		Err:  e,
	}
	select {
	case t.lineResultCh <- r:
	default:
	}
}

func (t *Terminal) screenSizeChanged(width, height int) {
	_ = t.rb.SetScreenWidth(width)
}

func (t *Terminal) opLineStart() {
	t.rb.MoveToLineStart()
}

func (t *Terminal) opBackward() {
	t.rb.MoveBackward()
}

func (t *Terminal) opDelete() {
	t.rb.Delete()
}

func (t *Terminal) opLineEnd() {
	t.rb.MoveToLineEnd()
}

func (t *Terminal) opForward() {
	t.rb.MoveForward()
}

func (t *Terminal) opBell() {

}

func (t *Terminal) opBackspace() {
	t.rb.Backspace()
}

func (t *Terminal) opTab() {

}

func (t *Terminal) opEnter() {
	t.rb.MoveToLineEnd()
	t.rb.WriteRune('\n')
	p := t.rb.Bytes()
	if len(p) > 0 {
		p = p[:len(p)-1]
	}
	t.sendLineResult(p, nil)
	t.rb.ResetBuf()
}

func (t *Terminal) opKill() {
	t.rb.Kill()
}

func (t *Terminal) opClear() {
	t.rb.Clear()
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
	t.rb.Transpose()
}

func (t *Terminal) opKillFront() {
	t.rb.KillFront()
}

func (t *Terminal) opYank() {
	t.rb.Yank()
}

func (t *Terminal) opBackEscapeWord() {
	t.rb.BackEscapeWord()
}
