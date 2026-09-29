package gen

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/govalues/decimal"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
)

// The JSON Schema formats generated value objects declare. They are JSON
// Schema formats where one exists (email, uri, uuid) and descriptive names
// where none does.
const (
	formatEmail   = "email"
	formatURI     = "uri"
	formatUUID    = "uuid"
	formatInt64   = "int64"
	formatDecimal = "decimal"
)

// integerPattern is the pattern of the text an int value object reads.
const integerPattern = `^[+-]?[0-9]+$`

// limitView is one optional limit of a schema, rendered as Go source.
type limitView struct {
	Set   bool
	Value string
}

// entryView is one key of the map a JSONSchema method returns, its value
// already rendered as Go source.
type entryView struct {
	Key, Value string
}

// schemaView is the JSONSchema method of one value object: the keywords it
// sets, in the order the template writes them.
type schemaView struct {
	// Format and Pattern are the plain text of the keywords, empty when unset.
	Format, Pattern                                          string
	Enum                                                     []string
	MinLength, MaxLength, Minimum, Maximum, ExclusiveMinimum limitView
	// FromMembers marks an enum value object, whose schema lists the members
	// of its catalogue instead of repeating their wire values, which the type
	// already spells twice.
	FromMembers bool
}

// unset is a limit that does not apply.
var unset = limitView{Set: false, Value: "0"}

// limitKeys lists the numeric keywords of a schema, in the order they are
// written, with the name of the constant that holds each one and whether it
// is a length (an int) or a number (a float64).
var limitKeys = []struct {
	keyword, constant string
	number            bool
}{
	{"minLength", "minimumLength", false}, {"maxLength", "maximumLength", false},
	{"minimum", "minimum", true}, {"maximum", "maximum", true}, {"exclusiveMinimum", "exclusiveMinimum", true},
}

// limit returns the limit a keyword of limitKeys is set from.
func (v *schemaView) limit(keyword string) limitView {
	switch keyword {
	case "minLength":
		return v.MinLength
	case "maxLength":
		return v.MaxLength
	case "minimum":
		return v.Minimum
	case "maximum":
		return v.Maximum
	default:
		return v.ExclusiveMinimum
	}
}

// Consts are the declarations of the limits that are set, named after the
// keyword they fill, so the method holds no magic number.
func (v *schemaView) Consts() []string {
	var out []string
	for _, key := range limitKeys {
		limit := v.limit(key.keyword)
		switch {
		case !limit.Set:
		case key.number:
			out = append(out, key.constant+" float64 = "+limit.Value)
		default:
			out = append(out, key.constant+" = "+limit.Value)
		}
	}
	return out
}

// Checkable reports whether a generated test can check a text against the
// schema: it sets a pattern, a length limit or an enum.
func (v *schemaView) Checkable() bool {
	return v.Pattern != "" || v.MinLength.Set || v.MaxLength.Set || v.Enum != nil
}

// Entries are the keywords the schema sets, in the order JSON Schema
// documents list them: type, format, pattern, enum, then the limits.
func (v *schemaView) Entries() []entryView {
	entries := []entryView{{Key: "type", Value: `"string"`}}
	if v.Format != "" {
		entries = append(entries, entryView{Key: "format", Value: strconv.Quote(v.Format)})
	}
	if v.Pattern != "" {
		entries = append(entries, entryView{Key: "pattern", Value: goLiteral(v.Pattern)})
	}
	if v.FromMembers {
		entries = append(entries, entryView{Key: "enum", Value: "values"})
	} else if v.Enum != nil {
		items := make([]string, len(v.Enum))
		for i, item := range v.Enum {
			items[i] = strconv.Quote(item)
		}
		entries = append(entries, entryView{Key: "enum", Value: "[]string{" + strings.Join(items, ", ") + "}"})
	}
	for _, key := range limitKeys {
		if v.limit(key.keyword).Set {
			entries = append(entries, entryView{Key: key.keyword, Value: key.constant})
		}
	}
	return entries
}

// goLiteral renders text as a Go string literal, raw when it can be written
// that way, which keeps a pattern readable as it was written.
func goLiteral(text string) string {
	if strings.ContainsAny(text, "`\r\n") || !utf8.ValidString(text) {
		return strconv.Quote(text)
	}
	return "`" + text + "`"
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
		MinLength: unset, MaxLength: unset, Minimum: unset, Maximum: unset, ExclusiveMinimum: unset,
	}

	switch d.Kind {
	case vogue.Int:
		v.Format, v.Pattern = formatInt64, integerPattern
	case vogue.Decimal:
		v.Format = formatDecimal
	case vogue.Enum:
		v.FromMembers = true
		for _, member := range d.Values {
			v.Enum = append(v.Enum, member.Value)
		}
	case vogue.ID:
		if d.Strategy == parse.IDInt64 {
			v.Format, v.Pattern = formatInt64, integerPattern
			v.ExclusiveMinimum = number("0")
		} else {
			v.Format = formatUUID
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

// apply narrows the schema by one rule of the directive.
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
			v.Pattern = use.Param
		}
	case "email":
		v.Format = formatEmail
	case "url":
		v.Format = formatURI
	case "uuid":
		v.Format = formatUUID
	case "oneof":
		v.Enum = enumItems(kind, use.Param)
	default:
	}
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
