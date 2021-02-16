package runeutil

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

func Copy(s []rune) []rune {
	result := make([]rune, len(s))
	copy(result, s)
	return result
}

func CopyAndGrow(s []rune, grow int) []rune {
	if grow < 0 {
		grow = 0
	}
	result := make([]rune, len(s), len(s)+grow)
	copy(result, s)
	return result
}

func EqualRune(a, b rune, fold bool) bool {
	if a == b {
		return true
	}
	if !fold {
		return false
	}
	if a > b {
		a, b = b, a
	}
	if b < utf8.RuneSelf && 'A' <= a && a <= 'Z' {
		if b == a+'a'-'A' {
			return true
		}
	}
	return false
}

func EqualRuneFold(a, b rune) bool {
	return EqualRune(a, b, true)
}

func EqualFold(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if EqualRuneFold(a[i], b[i]) {
			continue
		}
		return false
	}

	return true
}

func Equal(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Index(s []rune, r rune) int {
	for i := 0; i < len(s); i++ {
		if s[i] == r {
			return i
		}
	}
	return -1
}

// Search in runes from front to end
func IndexAll(s, sub []rune) int {
	return indexAll(s, sub, false)
}

func IndexAllFold(s, sub []rune) int {
	return indexAll(s, sub, true)
}

func indexAll(s, sub []rune, fold bool) int {
	for i := 0; i < len(s); i++ {
		found := true
		if len(s[i:]) < len(sub) {
			return -1
		}
		for j := 0; j < len(sub); j++ {
			if !EqualRune(s[i+j], sub[j], fold) {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}

// Search in runes from end to front
func IndexAllBck(s, sub []rune) int {
	return indexAllBck(s, sub, false)
}

func IndexAllBckFold(s, sub []rune) int {
	return indexAllBck(s, sub, true)
}

func indexAllBck(s, sub []rune, fold bool) int {
	for i := len(s) - len(sub); i >= 0; i-- {
		found := true
		for j := 0; j < len(sub); j++ {
			if !EqualRune(s[i+j], sub[j], fold) {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}
	return -1
}

func HasPrefix(s, prefix []rune) bool {
	if len(s) < len(prefix) {
		return false
	}
	return Equal(s[:len(prefix)], prefix)
}

func HasPrefixFold(s, prefix []rune) bool {
	if len(s) < len(prefix) {
		return false
	}
	return EqualFold(s[:len(prefix)], prefix)
}

func TrimSpaceLeft(s []rune) []rune {
	firstIndex := len(s)
	for i, r := range s {
		if unicode.IsSpace(r) == false {
			firstIndex = i
			break
		}
	}
	return s[firstIndex:]
}

var zeroWidth = []*unicode.RangeTable{
	unicode.Mn,
	unicode.Me,
	unicode.Cc,
	unicode.Cf,
}

var doubleWidth = []*unicode.RangeTable{
	unicode.Han,
	unicode.Hangul,
	unicode.Hiragana,
	unicode.Katakana,
}

func Width(r rune) int {
	if r == '\t' {
		return TabWidth
	}
	if unicode.IsOneOf(zeroWidth, r) {
		return 0
	}
	if unicode.IsOneOf(doubleWidth, r) {
		return 2
	}
	return 1
}

func WidthAll(s []rune) (length int) {
	for i := 0; i < len(s); i++ {
		length += Width(s[i])
	}
	return
}

func ColorFilter(s []rune) []rune {
	newr := make([]rune, 0, len(s))
	for pos := 0; pos < len(s); pos++ {
		if s[pos] == '\033' && s[pos+1] == '[' {
			idx := Index(s[pos+2:], 'm')
			if idx == -1 {
				continue
			}
			pos += idx + 2
			continue
		}
		newr = append(newr, s[pos])
	}
	return newr
}

func FillBackspace(s []rune) []byte {
	return bytes.Repeat([]byte{'\b'}, WidthAll(s))
}

func Aggregate(candicate [][]rune) (same []rune, size int) {
	for i := 0; i < len(candicate[0]); i++ {
		for j := 0; j < len(candicate)-1; j++ {
			if i >= len(candicate[j]) || i >= len(candicate[j+1]) {
				goto aggregate
			}
			if candicate[j][i] != candicate[j+1][i] {
				goto aggregate
			}
		}
		size = i + 1
	}
aggregate:
	if size > 0 {
		same = Copy(candicate[0][:size])
		for i := 0; i < len(candicate); i++ {
			n := Copy(candicate[i])
			copy(n, n[size:])
			candicate[i] = n[:len(n)-size]
		}
	}
	return
}

func SplitByLine(start, screenWidth int, s []rune) []string {
	var ret []string
	buf := bytes.NewBuffer(nil)
	currentWidth := start
	for _, r := range s {
		w := Width(r)
		currentWidth += w
		buf.WriteRune(r)
		if currentWidth >= screenWidth {
			ret = append(ret, buf.String())
			buf.Reset()
			currentWidth = 0
		}
	}
	ret = append(ret, buf.String())
	return ret
}

// calculate how many lines for N character
func LineCount(screenWidth, width int) int {
	result := width / screenWidth
	if width%screenWidth != 0 {
		result++
	}
	return result
}

func IsWordBreak(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
	case r >= 'A' && r <= 'Z':
	case r >= '0' && r <= '9':
	default:
		return true
	}
	return false
}
