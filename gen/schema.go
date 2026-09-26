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
		{"minLength", v.MinLength}, {"maxLength", v.MaxLength},
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
func newSchemaView(d parse.Directive) *schemaView {
	v := &schemaView{
		Format: `""`, Pattern: `""`,
		MinLength: unset, MaxLength: unset, Minimum: unset, Maximum: unset, ExclusiveMinimum: unset,
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

	for _, use := range d.Rules {
		v.apply(d.Kind, use)
	}
	return v
}

// apply narrows the schema by one rule of the directive.
func (v *schemaView) apply(kind vogue.Kind, use parse.RuleUse) {
	text := kind == vogue.String
	switch use.Rule.Name {
	case "required":
		if !v.MinLength.Set {
			v.MinLength = length("1")
		}
	case "len":
		v.MinLength, v.MaxLength = length(use.Param), length(use.Param)
	case "min":
		if text {
			v.MinLength = length(use.Param)
		} else {
			v.Minimum = number(use.Param)
		}
	case "max":
		if text {
			v.MaxLength = length(use.Param)
		} else {
			v.Maximum = number(use.Param)
		}
	case "positive":
		v.ExclusiveMinimum = number("0")
	case "nonneg":
		if !v.Minimum.Set {
			v.Minimum = number("0")
		}
	case "regex":
		v.Pattern = strconv.Quote(use.Param)
	case "email":
		v.Format = "schema.FormatEmail"
	case "url":
		v.Format = "schema.FormatURI"
	case "uuid":
		v.Format = "schema.FormatUUID"
	case "oneof":
		v.Enum = strings.Split(use.Param, ",")
	}
}

// length renders a length limit.
func length(param string) limitView { return limitView{Set: true, Value: param} }

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
