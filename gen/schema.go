package gen

import (
	"strconv"
	"strings"

	"github.com/govalues/decimal"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
)

// importSchema is the neutral schema package a JSONSchema method returns.
const importSchema = "github.com/MathiasHilgert/vogue/schema"

// limitView is one optional limit of a schema, rendered as Go source.
type limitView struct {
	Set   bool
	Value string
}

// schemaView is the JSONSchema method of one value object, every field
// already rendered as the Go source the template writes.
type schemaView struct {
	Format, Pattern                                          string
	Enum                                                     []string
	MinLength, MaxLength, Minimum, Maximum, ExclusiveMinimum limitView
	// modeled are the rules whose rejections the schema rejects too, so a
	// generated test may assert that a row they reject fails the schema.
	modeled map[string]bool
}

// unset is a limit that does not apply.
var unset = limitView{Set: false, Value: "0"}

// Consts are the declarations of the limits that are set, named after the
// field they fill, so the method compares no magic number.
func (v *schemaView) Consts() []string {
	var out []string
	for _, limit := range []struct {
		name string
		l    limitView
	}{
		{"minimumLength", v.MinLength}, {"maximumLength", v.MaxLength},
		{"minimum", v.Minimum}, {"maximum", v.Maximum}, {"exclusiveMinimum", v.ExclusiveMinimum},
	} {
		if limit.l.Set {
			out = append(out, limit.name+" = "+limit.l.Value)
		}
	}
	return out
}

// newSchemaView derives the schema of a directive from its kind and from the
// built-in rules it uses, which are recognised by name. A custom rule adds
// nothing: the generator cannot know what it accepts.
//
// The schema describes the canonical text String returns, so on a string only
// the checks written after the last normalizer contribute: a check that runs
// before `upper` constrains the input, not the stored value. Where several
// rules bound the same thing, the tightest bound wins, whatever order they
// are written in.
func newSchemaView(d parse.Directive) *schemaView {
	v := &schemaView{
		Format: `""`, Pattern: `""`,
		MinLength: unset, MaxLength: unset, Minimum: unset, Maximum: unset, ExclusiveMinimum: unset,
		modeled: map[string]bool{},
	}

	switch d.Kind {
	case vogue.Int:
		v.Format, v.Pattern = "schema.FormatInt64", "schema.IntegerPattern"
	case vogue.Decimal:
		v.Format = "schema.FormatDecimal"
	case vogue.Enum:
		for _, member := range d.Values {
			v.Enum = append(v.Enum, member.Value)
		}
	case vogue.ID:
		if d.Strategy == parse.IDInt64 {
			v.Format, v.Pattern = "schema.FormatInt64", "schema.IntegerPattern"
			v.ExclusiveMinimum = number("0")
		} else {
			v.Format = "schema.FormatUUID"
		}
	default:
	}

	for _, use := range afterLastNormalizer(d.Rules) {
		v.apply(d.Kind, use)
	}
	return v
}

// afterLastNormalizer returns the rules written after the last normalizer of
// a directive, which are the ones that see the value String returns.
func afterLastNormalizer(uses []parse.RuleUse) []parse.RuleUse {
	for i := len(uses) - 1; i >= 0; i-- {
		if uses[i].Rule.Normalize {
			return uses[i+1:]
		}
	}
	return uses
}

// apply narrows the schema by one rule of the directive, and records the rule
// as modeled when the schema now rejects what the rule rejects.
func (v *schemaView) apply(kind vogue.Kind, use parse.RuleUse) {
	text := kind == vogue.String
	switch use.Rule.Name {
	case "required":
		v.MinLength = atLeast(v.MinLength, "1")
	case "len":
		v.MinLength, v.MaxLength = atLeast(v.MinLength, use.Param), atMost(v.MaxLength, use.Param)
	case "min":
		if text {
			v.MinLength = atLeast(v.MinLength, use.Param)
		} else {
			v.Minimum = atLeast(v.Minimum, use.Param)
		}
	case "max":
		if text {
			v.MaxLength = atMost(v.MaxLength, use.Param)
		} else {
			v.Maximum = atMost(v.Maximum, use.Param)
		}
	case "positive":
		v.ExclusiveMinimum = atLeast(v.ExclusiveMinimum, "0")
	case "nonneg":
		v.Minimum = atLeast(v.Minimum, "0")
	case "regex":
		if ecmaCompatible(use.Param) {
			v.Pattern = strconv.Quote(use.Param)
		} else {
			return
		}
	case "email":
		v.Format = "schema.FormatEmail"
		return
	case "url":
		v.Format = "schema.FormatURI"
		return
	case "uuid":
		v.Format = "schema.FormatUUID"
		return
	case "oneof":
		v.Enum = enumItems(kind, use.Param)
	default:
		return
	}
	v.modeled[use.Rule.Name] = true
}

// enumItems returns the items of a oneof parameter as the value object writes
// them: an integer item in its canonical base-10 form.
func enumItems(kind vogue.Kind, param string) []string {
	items := strings.Split(param, ",")
	if kind != vogue.Int {
		return items
	}
	for i, item := range items {
		if parsed, err := strconv.ParseInt(item, 10, 64); err == nil {
			items[i] = strconv.FormatInt(parsed, 10)
		}
	}
	return items
}

// re2Only are the constructs of Go's RE2 syntax that an ECMA-262 regular
// expression — the dialect JSON Schema and OpenAPI patterns are written in —
// does not read the same way.
var re2Only = []string{`\A`, `\z`, `(?`, `\Q`, `\E`, `[[:`, `\C`, `\pL`, `\pN`, `\PL`, `\PN`, `\p{`, `\P{`}

// ecmaCompatible reports whether an RE2 pattern can be published unchanged as
// a JSON Schema pattern. One that uses an RE2-only construct — inline flags,
// \A and \z, POSIX classes, Unicode classes, \Q...\E — is left out of the
// schema rather than published in a dialect a client would misread.
func ecmaCompatible(pattern string) bool {
	for _, construct := range re2Only {
		if strings.Contains(pattern, construct) {
			return false
		}
	}
	return true
}

// atLeast returns the higher of a lower limit and a parameter.
func atLeast(limit limitView, param string) limitView {
	candidate := number(param)
	if !limit.Set || greater(candidate, limit) {
		return candidate
	}
	return limit
}

// atMost returns the lower of an upper limit and a parameter.
func atMost(limit limitView, param string) limitView {
	candidate := number(param)
	if !limit.Set || greater(limit, candidate) {
		return candidate
	}
	return limit
}

// greater reports whether a set limit is above another.
func greater(a, b limitView) bool {
	x, errX := decimal.Parse(a.Value)
	y, errY := decimal.Parse(b.Value)
	return errX == nil && errY == nil && x.Cmp(y) > 0
}

// number renders a numeric limit as a float literal, which is what the
// schema holds; a decimal bound is exact in the directive and approximate
// only here, where it is documentation.
func number(param string) limitView {
	d, err := decimal.Parse(param)
	if err != nil {
		return unset
	}
	f, _ := d.Float64()
	return limitView{Set: true, Value: strconv.FormatFloat(f, 'g', -1, 64)}
}
