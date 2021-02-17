package readline

import (
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"syscall"
	"unicode/utf8"
)

/*func correctErrNo0(e error) error {
	// errno 0 means everything is ok :)
	if e == nil {
		return nil
	}
	if errno, ok := e.(syscall.Errno); ok && errno == 0 {
		return nil
	}
	// if e.Error() == "errno 0" {
	// 	return nil
	// }
	return e
}*/

func isInterruptedSyscall(e error) bool {
	if errno, ok := e.(syscall.Errno); ok && errno == syscall.EINTR {
		return true
	}
	return false
	//return strings.Contains(e.Error(), "interrupted system call")
}

func checkBrokenPipe(w io.Writer) bool {
	_, err := w.Write([]byte{})
	return errors.Is(err, syscall.EPIPE)
}

func checkScreenBrokenPipe() bool {
	return checkBrokenPipe(os.Stdout) || checkBrokenPipe(os.Stderr)
}

func isFileChar(f *os.File) bool {
	fileInfo, err := f.Stat()
	return err == nil && (fileInfo.Mode()&os.ModeCharDevice) != 0
}

var (
	escapeRgx = regexp.MustCompile(`^(?P<esc>(?P<char>.)((?P<attr>\d+)(;(?P<attr2>\d+))?)?(?P<typ>[^\d;])?)?(?P<rem>.+)?$`)
)

type escapeKeyPair struct {
	Char       rune
	Attribute  int
	Attribute2 int
	Type       rune
	Remainder  []byte
}

func decodeEscapeKeyPair(p []byte) *escapeKeyPair {
	submatches := escapeRgx.FindSubmatch(p)
	p = submatches[escapeRgx.SubexpIndex("esc")]
	if len(p) <= 0 {
		return nil
	}
	result := &escapeKeyPair{
		Attribute:  -1,
		Attribute2: -1,
	}
	p = submatches[escapeRgx.SubexpIndex("char")]
	if len(p) > 0 {
		result.Char, _ = utf8.DecodeRune(p)
	}
	p = submatches[escapeRgx.SubexpIndex("attr")]
	if len(p) > 0 {
		i, err := strconv.ParseInt(string(p), 10, 32)
		if err != nil {
			i = -1
		}
		result.Attribute = int(i)
	}
	p = submatches[escapeRgx.SubexpIndex("attr2")]
	if len(p) > 0 {
		i, err := strconv.ParseInt(string(p), 10, 32)
		if err != nil {
			i = -1
		}
		result.Attribute2 = int(i)
	}
	p = submatches[escapeRgx.SubexpIndex("typ")]
	if len(p) > 0 {
		result.Type, _ = utf8.DecodeRune(p)
	}
	p = submatches[escapeRgx.SubexpIndex("rem")]
	if len(p) > 0 {
		result.Remainder = make([]byte, len(p))
		copy(result.Remainder, p)
	}
	return result
}

func encodeControlChars(p []byte) []byte {
	result := make([]byte, 0, len(p)*2)
	for _, b := range p {
		if b < 0x01F {
			result = append(result, '^', 0x40+b)
			continue
		}
		if b == CharBackspace {
			result = append(result, "^?"...)
			continue
		}
		result = append(result, b)
	}
	return result
}

type lineResult struct {
	Line []byte
	Err  error
}
