package vogue

import (
	"fmt"
	"strings"
)

// ParamPresence states whether a rule takes a parameter in the directive.
type ParamPresence uint8

const (
	// ParamNone means the rule is written without a parameter, as in `required`.
	ParamNone ParamPresence = iota
	// ParamRequired means the rule must be written as `name=param`.
	ParamRequired
	// ParamOptional means both `name` and `name=param` are accepted.
	ParamOptional
)

// String returns the directive-facing spelling of the presence.
func (p ParamPresence) String() string {
	switch p {
	case ParamNone:
		return "none"
	case ParamRequired:
		return "required"
	case ParamOptional:
		return "optional"
	default:
		return fmt.Sprintf("ParamPresence(%d)", uint8(p))
	}
}

// valid reports whether p is one of the declared presences.
func (p ParamPresence) valid() bool { return p <= ParamOptional }

// ParamType describes how a rule parameter is parsed and validated at generate
// time. It drives both the rejection of malformed directives and the wording of
// the resulting diagnostic.
type ParamType uint8

const (
	// paramTypeUnset is the zero value, only legal together with [ParamNone].
	paramTypeUnset ParamType = iota
	// ParamInt is a base-10 integer, as in `min=3`.
	ParamInt
	// ParamString is an opaque string, as in `prefix=SKU-`.
	ParamString
	// ParamList is a comma-separated list of strings, as in `oneof=a,b,c`.
	ParamList
	// ParamRegex is a regular expression compiled at generate time.
	ParamRegex
	// ParamNumber is a number whose Go type follows the kind the rule is used
	// on: a base-10 integer for the string and int kinds, where a bound counts
	// runes or compares an int64, and an exact decimal for the decimal kind.
	// It is what lets one `min` rule take `min=1` on a name and `min=0.5` on a
	// rate while still rejecting `min=0.5` on an integer.
	ParamNumber
)

// String returns the human-readable name of the parameter type, used in
// generator diagnostics.
func (t ParamType) String() string {
	switch t {
	case paramTypeUnset:
		return "unset"
	case ParamInt:
		return "int"
	case ParamString:
		return "string"
	case ParamList:
		return "list"
	case ParamRegex:
		return "regex"
	case ParamNumber:
		return "number"
	default:
		return fmt.Sprintf("ParamType(%d)", uint8(t))
	}
}

// valid reports whether t is one of the declared parameter types.
func (t ParamType) valid() bool { return t <= ParamNumber }

// ParamSpec declares the parameter contract of a rule. Its zero value means the
// rule takes no parameter.
type ParamSpec struct {
	// Presence states whether the parameter is absent, required or optional.
	Presence ParamPresence
	// Type states how the parameter is parsed, and must be set exactly when
	// Presence is not [ParamNone].
	Type ParamType
}

// String renders the spec as "none", or "<presence> <type>", for use in the
// rule catalogue and in generator diagnostics.
func (s ParamSpec) String() string {
	if s.Presence == ParamNone {
		return "none"
	}
	return s.Presence.String() + " " + s.Type.String()
}

// EmitContext carries the identifiers a rule needs to emit an inline check into
// generated code.
type EmitContext struct {
	// Var is the generated variable name holding the raw value under test.
	Var string
	// Param is the rule parameter as written in the directive, already
	// validated against the rule's [ParamSpec].
	Param string
	// Field is the value-object field name, available for checks that need to
	// name it.
	Field string
	// Kind is the value-object kind being generated, so a rule registered for
	// several kinds can emit the expression each one needs.
	Kind Kind
}

// FuncRef names a package-level function used as a rule check. The referenced
// function must have the signature `func(v T, param P) bool`, or `func(v T) bool`
// when the rule takes no parameter.
type FuncRef struct {
	// Path is the full import path of the package holding the function.
	Path string
	// Name is the exported function name.
	Name string
}

// Import returns the import path the generated file must add to call the
// function.
func (f FuncRef) Import() string { return f.Path }

// Selector returns the call expression `<package>.<Name>` as it appears in
// generated code. The package identifier is the last path element, skipping a
// trailing major-version element such as "/v2". An empty path yields the bare
// function name, which is how a rule refers to a helper emitted alongside it.
func (f FuncRef) Selector() string {
	if f.Path == "" {
		return f.Name
	}
	return packageIdent(f.Path) + "." + f.Name
}

// packageIdent derives the default package identifier from an import path.
func packageIdent(path string) string {
	parts := strings.Split(path, "/")
	last := parts[len(parts)-1]
	if len(parts) > 1 && isMajorVersion(last) {
		last = parts[len(parts)-2]
	}
	return last
}

// isMajorVersion reports whether a path element is a semantic-import-versioning
// suffix such as "v2".
func isMajorVersion(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for i := 1; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// Example is one input the test generator turns into a subtest case.
type Example struct {
	// Kinds are the value-object kinds the example is written for. A rule that
	// spans several kinds needs different inputs per kind: `min=1` is satisfied
	// by the string "a" and by the integer 1, and neither example is meaningful
	// for the other kind. The zero value is empty, which means "every kind the
	// rule applies to".
	Kinds KindSet
	// Param is the rule parameter the example is written against, empty when
	// the rule takes none. An example is only used for a directive that writes
	// exactly this parameter, because "too short" only means something relative
	// to a bound.
	Param string
	// In is the raw input handed to the generated constructor.
	In string
	// Note is the human-readable reason the example is what it is. The test
	// generator uses it to name the subtest; without one it names the subtest
	// after the rule and the input.
	Note string
}

// AppliesTo reports whether the example describes the behaviour of a rule used
// on kind with the parameter param. Empty [Example.Kinds] matches every kind,
// and the parameter must match exactly.
func (e Example) AppliesTo(kind Kind, param string) bool {
	return (e.Kinds.Empty() || e.Kinds.Has(kind)) && e.Param == param
}

// Normalization is one input a normalizer rule rewrites, together with the
// result. It is what lets the test generator prove that a normalizer actually
// normalizes instead of only asserting that its output still validates.
type Normalization struct {
	// In is the raw input handed to the generated constructor.
	In string
	// Out is the value the constructor is expected to hold afterwards, as its
	// String accessor reports it.
	Out string
	// Note is the human-readable reason the rewrite happens, used to name the
	// generated subtest.
	Note string
}

// Examples groups the accepted and rejected inputs of a rule. The test
// generator emits one subtest per example, which is what makes generated value
// objects arrive with tests at no authoring cost.
type Examples struct {
	// Valid inputs must be accepted by the rule.
	Valid []Example
	// Invalid inputs must be rejected by the rule.
	Invalid []Example
	// Normalized are the rewrites a rule with [Rule.Normalize] performs. They
	// are ignored for a checking rule, which rewrites nothing.
	Normalized []Normalization
}

// Rule is the definition of one validation tag. Rules are values, resolved at
// generate time: there is no runtime registry and no reflection. A project
// extends the catalogue by building its own generator binary that passes extra
// rules to the generator.
//
// Exactly one of [Rule.Emit] and [Rule.Call] must be set: a rule is either
// inlined into the generated constructor or delegated to a static function.
type Rule struct {
	// Name is the tag name as written in the directive, for example "min". It
	// must match ^[a-z][a-z0-9_]*$ so directives stay unambiguous to parse.
	Name string
	// Kinds are the value-object kinds the rule applies to. Applying a rule to
	// another kind is a generate-time error.
	Kinds KindSet
	// Doc is a one-paragraph description shown in the rule catalogue.
	Doc string
	// Param declares the parameter contract of the rule.
	Param ParamSpec
	// Message is the failure message template rendered with [MessageData].
	Message string
	// Normalize marks the rule as a normalizer rather than a check: its [Rule.Emit]
	// returns a Go statement that rewrites the working variable, and it produces
	// no failure. Normalizers run in the order they are written, so every rule
	// after them sees the rewritten value. A normalizer is always inlined, so it
	// may not use [Rule.Call].
	Normalize bool
	// Emit returns Go source. For an ordinary rule it is an inline boolean
	// expression that is true when the value is acceptable; for a rule with
	// [Rule.Normalize] set it is a statement assigning to [EmitContext.Var].
	Emit func(EmitContext) string
	// Call names a static function to call instead of emitting an expression.
	Call *FuncRef
	// Declare returns a package-level declaration the emitted expression needs,
	// such as a regular expression compiled once at process start rather than
	// on every constructor call. It is optional, requires [Rule.Emit], and is
	// emitted once per distinct declaration in a generated file: two directives
	// using the same rule with the same parameter share one declaration. The
	// returned source must therefore be self-contained and must name itself
	// deterministically from [EmitContext.Param], so that a rule declaring
	// `var x = ...` also emits an expression referring to that same `x`.
	Declare func(EmitContext) string
	// Imports are the import paths the emitted code needs, such as "strings"
	// for a rule whose expression calls strings.HasPrefix. A [Rule.Call] adds
	// its own package automatically and does not need to repeat it here.
	Imports []string
	// Examples feed the generated table-driven tests.
	Examples Examples
}

// Validate reports whether the rule definition is internally consistent. The
// generator calls it before using a rule, so a malformed custom rule fails at
// generate time with a message naming the offending field rather than
// producing code that does not compile.
func (r Rule) Validate() error {
	if r.Name == "" {
		return fmt.Errorf("vogue: rule name must not be empty")
	}
	if !isTagName(r.Name) {
		return fmt.Errorf("vogue: rule %q is not a valid tag name (want ^[a-z][a-z0-9_]*$)", r.Name)
	}
	if r.Kinds.Empty() {
		return fmt.Errorf("vogue: rule %q must declare at least one kind", r.Name)
	}
	if r.Message == "" {
		return fmt.Errorf("vogue: rule %q message must not be empty", r.Name)
	}
	if _, err := compileMessage(r.Message); err != nil {
		return fmt.Errorf("vogue: rule %q: %w", r.Name, err)
	}
	if (r.Emit == nil) == (r.Call == nil) {
		return fmt.Errorf("vogue: rule %q: exactly one of Emit or Call must be set", r.Name)
	}
	for _, path := range r.Imports {
		if path == "" {
			return fmt.Errorf("vogue: rule %q: import path must not be empty", r.Name)
		}
	}
	if r.Declare != nil && r.Emit == nil {
		return fmt.Errorf("vogue: rule %q: Declare requires Emit, a call-backed rule declares nothing", r.Name)
	}
	if r.Call != nil {
		if r.Normalize {
			return fmt.Errorf("vogue: rule %q: a normalizing rule must emit a statement, Call is not supported", r.Name)
		}
		if r.Call.Path == "" {
			return fmt.Errorf("vogue: rule %q: call path must not be empty", r.Name)
		}
		if r.Call.Name == "" {
			return fmt.Errorf("vogue: rule %q: call name must not be empty", r.Name)
		}
	}
	return r.validateParam()
}

// validateParam checks the parameter contract in isolation.
func (r Rule) validateParam() error {
	if !r.Param.Presence.valid() {
		return fmt.Errorf("vogue: rule %q: unknown param presence %d", r.Name, uint8(r.Param.Presence))
	}
	if !r.Param.Type.valid() {
		return fmt.Errorf("vogue: rule %q: unknown param type %d", r.Name, uint8(r.Param.Type))
	}
	if r.Param.Presence == ParamNone {
		if r.Param.Type != paramTypeUnset {
			return fmt.Errorf("vogue: rule %q: param type must not be set when the rule takes no parameter", r.Name)
		}
		return nil
	}
	if r.Param.Type == paramTypeUnset {
		return fmt.Errorf("vogue: rule %q: param type must be set when the parameter is %s", r.Name, r.Param.Presence)
	}
	return nil
}

// Summary returns the first sentence of [Rule.Doc], with the line breaks of a
// wrapped documentation collapsed. It is what a one-line catalogue entry shows
// — the `-list` table of the command, the rule table of the README — so the
// first sentence of a rule's documentation should say what it rejects and the
// rest may go into detail.
func (r Rule) Summary() string {
	doc := strings.Join(strings.Fields(r.Doc), " ")
	if end := strings.Index(doc, ". "); end >= 0 {
		return doc[:end+1]
	}
	return doc
}

// RenderMessage renders the rule message with the given data.
func (r Rule) RenderMessage(data MessageData) (string, error) {
	return RenderMessage(r.Message, data)
}

// isTagName reports whether s matches ^[a-z][a-z0-9_]*$. It is hand-written
// rather than a regexp so rule validation stays allocation-free.
func isTagName(s string) bool {
	if s == "" || s[0] < 'a' || s[0] > 'z' {
		return false
	}
	for i := 1; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '_':
		default:
			return false
		}
	}
	return true
}

// RuleSet is an ordered, name-unique collection of rules. The zero value is an
// empty set ready to use. Insertion order is preserved so the catalogue and the
// generator diagnostics read in the order the author declared.
//
// A RuleSet is not safe for concurrent mutation; build it once at startup and
// read it afterwards.
type RuleSet struct {
	rules []Rule
	index map[string]int
}

// Add validates and appends the rules, preserving order. Nothing is added when
// any rule is invalid or duplicates a name already present, so a failed Add
// leaves the set exactly as it was.
func (s *RuleSet) Add(rules ...Rule) error {
	seen := make(map[string]struct{}, len(rules))
	for _, r := range rules {
		if err := r.Validate(); err != nil {
			return err
		}
		if i, ok := s.index[r.Name]; ok {
			return fmt.Errorf("vogue: duplicate rule %q: already registered for kinds [%s], cannot add another for kinds [%s]",
				r.Name, s.rules[i].Kinds, r.Kinds)
		}
		if _, ok := seen[r.Name]; ok {
			return fmt.Errorf("vogue: duplicate rule %q: declared twice in the same Add", r.Name)
		}
		seen[r.Name] = struct{}{}
	}

	if s.index == nil {
		s.index = make(map[string]int, len(rules))
	}
	for _, r := range rules {
		s.index[r.Name] = len(s.rules)
		s.rules = append(s.rules, r)
	}
	return nil
}

// Get returns the rule registered under name.
func (s *RuleSet) Get(name string) (Rule, bool) {
	i, ok := s.index[name]
	if !ok {
		return Rule{}, false
	}
	return s.rules[i], true
}

// Len returns the number of registered rules.
func (s *RuleSet) Len() int { return len(s.rules) }

// Names returns the registered rule names in insertion order.
func (s *RuleSet) Names() []string {
	out := make([]string, len(s.rules))
	for i := range s.rules {
		out[i] = s.rules[i].Name
	}
	return out
}

// Rules returns a copy of the registered rules in insertion order.
func (s *RuleSet) Rules() []Rule {
	out := make([]Rule, len(s.rules))
	copy(out, s.rules)
	return out
}

// Suggest returns the registered name closest to the given one, when the
// Damerau-Levenshtein distance between them is at most two. It powers the
// "did you mean" part of an unknown-rule diagnostic.
//
// Ties are broken alphabetically, so the same directive always produces the
// same suggestion regardless of registration order.
func (s *RuleSet) Suggest(name string) (string, bool) {
	const maxDistance = 2

	best := ""
	bestDistance := maxDistance + 1
	for i := range s.rules {
		candidate := s.rules[i].Name
		d := damerauLevenshtein(name, candidate)
		if d > maxDistance {
			continue
		}
		if d < bestDistance || (d == bestDistance && candidate < best) {
			best, bestDistance = candidate, d
		}
	}
	return best, best != ""
}

// damerauLevenshtein returns the optimal string alignment distance between a
// and b, counting insertions, deletions, substitutions and transpositions of
// adjacent characters as one edit each.
func damerauLevenshtein(a, b string) int {
	if a == b {
		return 0
	}
	ar, br := []rune(a), []rune(b)

	prev2 := make([]int, len(br)+1)
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
			if i > 1 && j > 1 && ar[i-1] == br[j-2] && ar[i-2] == br[j-1] {
				curr[j] = min(curr[j], prev2[j-2]+1)
			}
		}
		prev2, prev, curr = prev, curr, prev2
	}
	return prev[len(br)]
}
