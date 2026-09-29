package parse

import (
	"go/token"
	"strconv"

	"github.com/MathiasHilgert/vogue"
)

// Package is the validated directive model of one Go package. It is the whole
// input of the code and test generators: everything the templates need has
// already been resolved and checked here.
type Package struct {
	// Path is the directory the package was read from, empty when the package
	// was parsed from memory.
	Path string
	// Name is the Go package name the generated files belong to.
	Name string
	// Files holds one entry per parsed source file, in read order.
	Files []File
}

// Directives returns every directive of the package in file order, which is the
// order the generator emits value objects in.
func (p *Package) Directives() []Directive {
	var out []Directive
	for i := range p.Files {
		out = append(out, p.Files[i].Directives...)
	}
	return out
}

// File is one source file and the directives found in it.
type File struct {
	// Path is the file name as the parser saw it.
	Path string
	// Directives are the value objects declared in the file, in source order.
	Directives []Directive
}

// IDStrategy selects how an id value object mints and stores its value. It is
// the optional token written after the name of an `//vogue:id` directive.
type IDStrategy uint8

const (
	// IDUUIDv7 mints time-ordered UUIDv7 identifiers. It is the default, so it
	// is the zero value of the type.
	IDUUIDv7 IDStrategy = iota
	// IDUUIDv4 mints random UUIDv4 identifiers.
	IDUUIDv4
	// IDInt64 wraps a database-assigned 64-bit integer.
	IDInt64
)

// idStrategyNames maps a strategy to its directive spelling. The slice index is
// the strategy value, so it doubles as the set of known strategies.
var idStrategyNames = [...]string{
	IDUUIDv7: "uuid7",
	IDUUIDv4: "uuid4",
	IDInt64:  "int64",
}

// String returns the directive spelling of the strategy, or an
// IDStrategy(<n>) placeholder when the receiver is not one of them.
func (s IDStrategy) String() string {
	if int(s) < len(idStrategyNames) {
		return idStrategyNames[s]
	}
	return "IDStrategy(" + strconv.Itoa(int(s)) + ")"
}

// parseIDStrategy resolves the directive spelling of a strategy.
func parseIDStrategy(s string) (IDStrategy, bool) {
	for i, name := range idStrategyNames {
		if name == s {
			return IDStrategy(i), true //nolint:gosec // the index is bounded by the array.
		}
	}
	return 0, false
}

// Directive is one validated `//vogue:` line: a value object to generate.
type Directive struct {
	// Pos is the position of the directive comment.
	Pos token.Position
	// Kind is the value-object shape to generate.
	Kind vogue.Kind
	// Name is the Go type name of the value object.
	Name string
	// Field is the lower-camel name used in field errors and JSON.
	Field string
	// Doc holds the ordinary comment lines written directly above the
	// directive, without their slashes. It may be empty.
	Doc string
	// Rules are the validation rules, in the order written, which is also the
	// order the generated constructor evaluates them in. It is empty for the
	// enum and id kinds.
	Rules []RuleUse
	// Examples are the values the directive declares valid with
	// `example=<value>` tokens, in the order written. The generated test
	// requires the constructor to accept every one of them and proves the
	// round trips with the first, which is how a directive whose rules
	// declare no example that survives all of them still gets a tested
	// sample. It is empty for the enum and id kinds.
	Examples []string
	// RegexMessage replaces the generic message of the `regex` rule, written
	// with a `regex_message="..."` token. It is empty when none was written
	// and only legal on a directive that uses `regex`.
	RegexMessage string
	// Values are the members of an enum, in the order written. It is empty for
	// every other kind.
	Values []EnumValue
	// Strategy selects how an id value object mints its value. It is only
	// meaningful for the id kind, where its zero value is [IDUUIDv7].
	Strategy IDStrategy
	// Catalogue is the name of the empty struct type whose methods expose the
	// members of an enum, the plural of [Directive.Name]: PlaceKind has
	// PlaceKinds, TabStatus has TabStatuses. It is empty for every other kind.
	Catalogue string
}

// RuleUse is one rule as applied by a directive.
type RuleUse struct {
	// Rule is the definition the token resolved to.
	Rule vogue.Rule
	// Param is the parameter as written, already unquoted and validated
	// against the rule's [vogue.ParamSpec]. It is empty when the rule takes no
	// parameter.
	Param string
	// Pos is the position of the rule token, so a later generator stage can
	// report against the exact token.
	Pos token.Position
}

// EnumValue is one member of an enum value object.
type EnumValue struct {
	// Value is the wire value as written in the directive.
	Value string
	// Const is the prefixed Go name of the member, such as TabStatusOpen. The
	// generator no longer declares it; it is kept for tools that name members
	// the way vogue 0.1 did.
	Const string
	// Method is the name of the catalogue method returning the member, such as
	// Open for TabStatuses{}.Open().
	Method string
}
