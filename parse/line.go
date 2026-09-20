package parse

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/MathiasHilgert/vogue"
)

// prefix introduces a vogue directive. It is written without a space so an
// ordinary doc comment can never be mistaken for one.
const prefix = "//vogue:"

// kindNames lists the directive spellings of every kind, for diagnostics.
var kindNames = []string{
	vogue.String.String(),
	vogue.Int.String(),
	vogue.Enum.String(),
	vogue.ID.String(),
	vogue.Decimal.String(),
}

// dtoken is one whitespace-separated directive token together with its byte
// offset in the comment line, which is what turns a token into a column.
type dtoken struct {
	text string
	off  int
}

// Line tokenizes one directive comment. It returns the kind, the value-object
// name and the remaining tokens, with quoted parameters already unquoted.
//
// It is exported so the generator can reuse the exact same tokenization when it
// needs to quote a directive back to the author.
func Line(line string) (kind vogue.Kind, name string, tokens []string, err error) {
	k, n, rest, lerr := parseLine(line)
	if lerr != nil {
		return 0, "", nil, lerr
	}
	for _, t := range rest {
		tokens = append(tokens, t.text)
	}
	return k, n.text, tokens, nil
}

// parseLine is the positional form of [Line]: it keeps the byte offset of every
// token so the caller can build precise diagnostics.
func parseLine(line string) (vogue.Kind, dtoken, []dtoken, *Error) {
	if !strings.HasPrefix(strings.TrimSpace(line), prefix) {
		return 0, dtoken{}, nil, &Error{Msg: "not a vogue directive"}
	}
	start := strings.Index(line, prefix) + len(prefix)

	tokens, err := tokenize(line, start)
	if err != nil {
		return 0, dtoken{}, nil, err
	}
	if len(tokens) == 0 {
		return 0, dtoken{}, nil, &Error{Msg: fmt.Sprintf("missing kind after %q", prefix)}
	}

	kindTok := tokens[0]
	if kindTok.off != start {
		return 0, dtoken{}, nil, &Error{Msg: fmt.Sprintf("missing kind after %q", prefix)}
	}
	kind, kerr := vogue.ParseKind(kindTok.text)
	if kerr != nil {
		e := &Error{Msg: fmt.Sprintf("unknown kind %q", kindTok.text)}
		if near, ok := nearest(kindTok.text, kindNames); ok {
			e.Hint = fmt.Sprintf("did you mean %q?", near)
		} else {
			e.Hint = "want one of: " + strings.Join(kindNames, ", ")
		}
		return 0, dtoken{}, nil, e
	}
	if len(tokens) == 1 {
		return 0, dtoken{}, nil, &Error{Msg: fmt.Sprintf("missing type name after kind %q", kind)}
	}
	return kind, tokens[1], tokens[2:], nil
}

// tokenize splits the line from the given offset into whitespace-separated
// tokens. A double-quoted run is taken literally and unquoted with Go escape
// rules, which is how a parameter carries spaces or an equals sign.
func tokenize(line string, from int) ([]dtoken, *Error) {
	var tokens []dtoken
	i := from
	for i < len(line) {
		for i < len(line) && isSpace(line[i]) {
			i++
		}
		if i >= len(line) {
			break
		}

		var (
			b     strings.Builder
			begin = i
		)
		for i < len(line) && !isSpace(line[i]) {
			if line[i] != '"' {
				b.WriteByte(line[i])
				i++
				continue
			}
			end, ok := closingQuote(line, i)
			if !ok {
				return nil, &Error{Msg: "unterminated quoted parameter"}
			}
			raw := line[i : end+1]
			unquoted, uerr := strconv.Unquote(raw)
			if uerr != nil {
				return nil, &Error{Msg: fmt.Sprintf("invalid quoted parameter %q", raw)}
			}
			b.WriteString(unquoted)
			i = end + 1
		}
		tokens = append(tokens, dtoken{text: b.String(), off: begin})
	}
	return tokens, nil
}

// closingQuote returns the index of the quote closing the one at open,
// honouring backslash escapes.
func closingQuote(line string, open int) (int, bool) {
	for i := open + 1; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '"':
			return i, true
		}
	}
	return 0, false
}

// isSpace reports whether the byte separates directive tokens.
func isSpace(c byte) bool { return c == ' ' || c == '\t' }

// nearest returns the candidate closest to s when their edit distance is at
// most two, which is the threshold vogue uses for "did you mean" hints.
func nearest(s string, candidates []string) (string, bool) {
	const maxDistance = 2

	best, bestDistance := "", maxDistance+1
	for _, c := range candidates {
		d := distance(s, c)
		if d > maxDistance {
			continue
		}
		if d < bestDistance || (d == bestDistance && c < best) {
			best, bestDistance = c, d
		}
	}
	return best, best != ""
}

// distance returns the Levenshtein distance between a and b.
func distance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}
