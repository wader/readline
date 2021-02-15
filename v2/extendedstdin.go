package readline

import (
	"bytes"
	"io"
	"sync"
)

// extendedStdin is a stdin reader which can prepend some data before
// reading into the real stdin.
type extendedStdin struct {
	stdin          io.Reader
	pipe1Reader    *io.PipeReader
	pipe2Reader    *io.PipeReader
	pipe2Writer    *io.PipeWriter
	wg             sync.WaitGroup
	mu             sync.Mutex
	buf            *bytes.Buffer
	pipe1ReaderErr error
}

// newExtendedStdin gives you extendedStdin
func newExtendedStdin(stdin io.Reader) (io.ReadCloser, io.Writer) {
	r1, w1 := io.Pipe()
	r2, w2 := io.Pipe()
	s := &extendedStdin{
		stdin:       stdin,
		pipe1Reader: r1,
		pipe2Reader: r2,
		pipe2Writer: w2,
		buf:         bytes.NewBuffer(make([]byte, 0, 4096)),
	}
	s.wg.Add(1)
	go s.pipe1Loop()
	go s.pipe2Loop()
	return s, w1
}

func (s *extendedStdin) pipe1Loop() {
	defer s.wg.Done()
	buf := make([]byte, 4096)
	for {
		var err error
		var n int
		n, err = s.pipe1Reader.Read(buf)
		s.mu.Lock()
		if n > 0 {
			_, _ = s.buf.Write(buf[:n])
		}
		s.pipe1ReaderErr = err
		s.mu.Unlock()
		if err != nil {
			break
		}
	}
}

func (s *extendedStdin) pipe2Loop() {
	buf := make([]byte, 4096)
	for {
		var err error
		var n int
		n, err = s.stdin.Read(buf)
		var werr error
		if n > 0 {
			_, werr = s.pipe2Writer.Write(buf[:n])
		}
		if err != nil {
			_ = s.pipe2Writer.CloseWithError(err)
			break
		}
		if werr != nil {
			break
		}
	}
}

// Read will read from the local buffer and if no data, read from stdin
func (s *extendedStdin) Read(p []byte) (n int, err error) {
	s.mu.Lock()
	n, err = s.buf.Read(p)
	if err == nil || s.pipe1ReaderErr != nil {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	return s.pipe2Reader.Read(p)
}

func (s *extendedStdin) Close() error {
	_ = s.pipe1Reader.Close()
	_ = s.pipe2Writer.Close()
	return nil
}
