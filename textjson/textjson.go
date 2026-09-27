// Package textjson writes and reads the JSON string a generated value object
// crosses a JSON boundary as, without encoding/json.
//
// A generated value object implements json.Marshaler so that its zero value
// is null rather than the text of an empty or zero value, which would read
// back as a constructed one. Implementing it needs a JSON string encoder, and
// a hexagonal domain package is commonly forbidden to import encoding/json,
// so the generated methods call this package instead. It imports nothing but
// the standard library's unicode/utf8, strconv and errors.
package textjson

import (
	"errors"
	"strconv"
	"unicode/utf16"
	"unicode/utf8"
)

// ErrNotString is returned by [Unquote] for JSON that is neither a string nor
// null.
var ErrNotString = errors.New("textjson: not a JSON string or null")

// hexDigits spells the \u escapes Quote writes.
const hexDigits = "0123456789abcdef"

// Null returns the JSON null literal.
func Null() []byte { return []byte("null") }

// Quote returns text as a JSON string. It escapes what encoding/json escapes
// — quotes, backslashes, control characters, U+2028, U+2029 and the HTML
// characters <, > and & — and replaces invalid UTF-8 with U+FFFD, so the
// result is byte for byte what encoding/json would write.
func Quote(text []byte) []byte {
	out := make([]byte, 0, len(text)+2)
	out = append(out, '"')
	for index := 0; index < len(text); {
		character, size := utf8.DecodeRune(text[index:])
		index += size
		switch {
		case character == utf8.RuneError && size == 1:
			out = append(out, `�`...)
		case character == '"' || character == '\\':
			out = append(out, '\\', byte(character))
		case character == '\n':
			out = append(out, `\n`...)
		case character == '\r':
			out = append(out, `\r`...)
		case character == '\t':
			out = append(out, `\t`...)
		case character < 0x20, character == '<', character == '>', character == '&',
			character == ' ', character == ' ':
			out = append(out, '\\', 'u',
				hexDigits[character>>12&0xf], hexDigits[character>>8&0xf],
				hexDigits[character>>4&0xf], hexDigits[character&0xf])
		default:
			out = utf8.AppendRune(out, character)
		}
	}
	return append(out, '"')
}

// Unquote reads a JSON string, or null. It reports isNull for null, and
// [ErrNotString] for anything else that is not a well-formed JSON string.
func Unquote(data []byte) (text []byte, isNull bool, err error) {
	if string(data) == "null" {
		return nil, true, nil
	}
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return nil, false, ErrNotString
	}
	body := data[1 : len(data)-1]
	out := make([]byte, 0, len(body))
	for index := 0; index < len(body); {
		character := body[index]
		switch {
		case character == '"' || character < 0x20:
			return nil, false, ErrNotString
		case character != '\\':
			out = append(out, character)
			index++
			continue
		}
		decoded, consumed, ok := unescape(body[index:])
		if !ok {
			return nil, false, ErrNotString
		}
		out = utf8.AppendRune(out, decoded)
		index += consumed
	}
	if !utf8.Valid(out) {
		out = []byte(string([]rune(string(out))))
	}
	return out, false, nil
}

// unescape reads one escape sequence at the start of escaped, returning the
// rune it stands for and how many bytes it took.
func unescape(escaped []byte) (rune, int, bool) {
	if len(escaped) < 2 {
		return 0, 0, false
	}
	switch escaped[1] {
	case '"', '\\', '/':
		return rune(escaped[1]), 2, true
	case 'b':
		return '\b', 2, true
	case 'f':
		return '\f', 2, true
	case 'n':
		return '\n', 2, true
	case 'r':
		return '\r', 2, true
	case 't':
		return '\t', 2, true
	case 'u':
		return unescapeUnicode(escaped)
	default:
		return 0, 0, false
	}
}

// unescapeUnicode reads a \uXXXX escape, joining a surrogate pair written as
// two escapes into the one rune it encodes.
func unescapeUnicode(escaped []byte) (rune, int, bool) {
	first, ok := hex4(escaped[2:])
	if !ok {
		return 0, 0, false
	}
	if !utf16.IsSurrogate(first) {
		return first, 6, true
	}
	if len(escaped) >= 12 && escaped[6] == '\\' && escaped[7] == 'u' {
		if second, ok := hex4(escaped[8:]); ok {
			if joined := utf16.DecodeRune(first, second); joined != utf8.RuneError {
				return joined, 12, true
			}
		}
	}
	return utf8.RuneError, 6, true
}

// hex4 parses four hexadecimal digits.
func hex4(digits []byte) (rune, bool) {
	if len(digits) < 4 {
		return 0, false
	}
	value, err := strconv.ParseUint(string(digits[:4]), 16, 32)
	return rune(value), err == nil
}
