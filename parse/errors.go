package parse

import (
	"go/token"
	"sort"
	"strings"
)

// Error is one directive diagnostic. It carries the source position of the
// offending token so the message reads like a compiler error:
//
//	file.go:12:1: unknown rule "mni" for kind string (did you mean "min"?)
//
// Hint is optional and holds the actionable half of the message.
type Error struct {
	// Pos is the position of the offending token. It is the zero value for
	// diagnostics produced outside a file, such as those returned by [Line].
	Pos token.Position
	// Msg states what is wrong.
	Msg string
	// Hint states how to fix it, and is empty when there is nothing to add.
	Hint string
}

// Error renders the diagnostic, prefixing the position when there is one and
// appending the hint in parentheses when there is one.
func (e *Error) Error() string {
	var b strings.Builder
	if e.Pos.IsValid() || e.Pos.Filename != "" {
		b.WriteString(e.Pos.String())
		b.WriteString(": ")
	}
	b.WriteString(e.Msg)
	if e.Hint != "" {
		b.WriteString(" (")
		b.WriteString(e.Hint)
		b.WriteString(")")
	}
	return b.String()
}

// Errors is the full set of diagnostics found in a package. The parser reports
// every problem it finds rather than stopping at the first, so one run of the
// generator surfaces every broken directive.
type Errors []*Error

// Error renders one diagnostic per line, in position order.
func (e Errors) Error() string {
	parts := make([]string, len(e))
	for i, err := range e {
		parts[i] = err.Error()
	}
	return strings.Join(parts, "\n")
}

// sort orders the diagnostics by file, then line, then column, so the output is
// stable regardless of the order in which the parser found them.
func (e Errors) sort() {
	sort.SliceStable(e, func(i, j int) bool {
		a, b := e[i].Pos, e[j].Pos
		switch {
		case a.Filename != b.Filename:
			return a.Filename < b.Filename
		case a.Line != b.Line:
			return a.Line < b.Line
		default:
			return a.Column < b.Column
		}
	})
}

// err appends a diagnostic without a hint.
func (e *Errors) err(pos token.Position, msg string) {
	*e = append(*e, &Error{Pos: pos, Msg: msg})
}

// hint appends a diagnostic with a hint.
func (e *Errors) hint(pos token.Position, msg, hint string) {
	*e = append(*e, &Error{Pos: pos, Msg: msg, Hint: hint})
}
