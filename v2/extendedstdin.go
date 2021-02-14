package readline

import (
	"bytes"
	"io"
	"sync"
)

// extendedStdin is a stdin reader which can prepend some data before
// reading into the real stdin.
type extendedStdin struct {
	wg          sync.WaitGroup
	mu          sync.Mutex
	stdin       io.Reader
	stdinBuf    io.ReadCloser
	stdinBufErr error
	buf         *bytes.Buffer
}

// newExtendedStdin gives you extendedStdin
func newExtendedStdin(stdin io.Reader) (io.ReadCloser, io.Writer) {
	r, w := io.Pipe()
	s := &extendedStdin{
		stdin:    stdin,
		stdinBuf: r,
		buf:      bytes.NewBuffer(make([]byte, 0, 4096)),
	}
	s.wg.Add(1)
	go s.ioloop()
	return s, w
}

func (s *extendedStdin) ioloop() {
	defer s.wg.Done()
	buf := make([]byte, 1024)
	for {
		var err error
		var n int
		n, err = s.stdinBuf.Read(buf)
		s.mu.Lock()
		_, _ = s.buf.Write(buf[:n])
		s.stdinBufErr = err
		s.mu.Unlock()
		if err != nil {
			break
		}
	}
}

// Read will read from the local buffer and if no data, read from stdin
func (s *extendedStdin) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	n, err = s.buf.Read(p)
	if err == nil || s.stdinBufErr != nil {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	return s.stdin.Read(p)
}

func (s *extendedStdin) Close() error {
	_ = s.stdinBuf.Close()
	return nil
}
