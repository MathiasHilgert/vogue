package gen

import (
	"fmt"
	"go/token"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/MathiasHilgert/vogue"
)

// methodView is one unexported method the rules of a generated type asked for
// through [vogue.Rule.Method].
type methodView struct {
	// Name is the method name and Rule the rule it implements.
	Name, Rule string
	// ParamType is the Go type of the value parameter, which follows the kind.
	ParamType string
	// Body is the statements of the method.
	Body string
	// Long marks a body that, with its signature and closing brace, is longer
	// than [longMethodLines], and so carries a directive for funlen.
	Long bool
}

// longMethodLines is the length golangci-lint's funlen accepts by default. A
// method above it — the timezone switch lists every zone of the database — is
// written with a targeted directive, so a strict profile does not fail on
// what is data rather than logic.
const longMethodLines = 60

// isLong reports whether a method with this body is longer than
// [longMethodLines], counting its signature and its closing brace.
func isLong(body string) bool {
	const signatureAndBrace = 2
	return strings.Count(body, "\n")+1+signatureAndBrace > longMethodLines
}

// fieldNames are the fields of a generated struct, which a method may not
// take the name of.
var fieldNames = map[string]struct{}{"value": {}, "set": {}}

// methodSet collects the methods of one generated type in first-seen order,
// which keeps the generated file byte-stable across runs. A method is written
// once per type: a rule reaching the same name with the same body twice shares
// one method.
type methodSet struct {
	byName map[string]string
	order  []methodView
}

// newMethodSet returns an empty set.
func newMethodSet() *methodSet {
	return &methodSet{byName: map[string]string{}}
}

// add records a method, refusing a name two different bodies claim.
func (s *methodSet) add(m methodView) error {
	if existing, taken := s.byName[m.Name]; taken {
		if existing == m.Body {
			return nil
		}
		return fmt.Errorf("rule %q: method %q is already written for this type with a different body", m.Rule, m.Name)
	}
	s.byName[m.Name] = m.Body
	s.order = append(s.order, m)
	return nil
}

// all returns the recorded methods in first-seen order.
func (s *methodSet) all() []methodView { return s.order }

// checkMethodName reports why name cannot be the name of a generated method.
func checkMethodName(rule, name string) error {
	first, _ := utf8.DecodeRuneInString(name)
	switch {
	case !token.IsIdentifier(name):
		return fmt.Errorf("rule %q: method name %q is not a Go identifier", rule, name)
	case !unicode.IsLower(first) && first != '_':
		return fmt.Errorf("rule %q: method name %q must be unexported", rule, name)
	}
	if _, taken := fieldNames[name]; taken {
		return fmt.Errorf("rule %q: method name %q collides with a field of the generated type", rule, name)
	}
	return nil
}

// methodParamType returns the Go type of the value a method receives, which
// is the type of the working value of the constructor for the kind.
func methodParamType(kind vogue.Kind) string {
	switch kind {
	case vogue.Int:
		return "int64"
	case vogue.Decimal:
		return "decimal.Decimal"
	default:
		return "string"
	}
}
