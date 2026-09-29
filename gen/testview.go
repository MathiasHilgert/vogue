package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"slices"
	"strconv"
	"strings"

	"github.com/govalues/decimal"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
)

// Import paths only the generated tests need.
const (
	importTesting = "testing"
	importRegexp  = "regexp"
	importSlices  = "slices"
	importUTF8    = "unicode/utf8"
)

// caseView is one rejected row of a generated table: what the row says about
// the input, the input, and the rules expected to reject it.
type caseView struct {
	Note  string
	Lit   string
	Rules []string
}

// normCaseView is one rewrite a normalizer performs, as the generated test
// states it: feed In to the constructor and expect the value object built from
// Out.
type normCaseView struct {
	Name    string
	In, Out string
}

// testView is the data a test template renders. As with [voView], the fields
// that do not apply to the kind being rendered stay at their zero value.
type testView struct {
	// Name is the value-object type and Field the name it reports failures
	// under.
	Name, Field string

	// InType is the Go type the table feeds Ctor, and Verb the fmt verb that
	// prints one of its values in a failure message.
	InType, Verb string
	// Ctor is the constructor the table drives. It is New<Name> for every kind
	// whose constructor takes a Go literal the table can write; the decimal
	// kind takes a decimal.Decimal, which no literal can spell, so its table
	// is written in the text New<Name>FromString reads.
	Ctor string
	// Read is the accessor returning the value as Ctor took it, empty when an
	// accepted input may legitimately be held differently.
	Read string
	// FromString is the textual constructor the tests prove, and ParseRule
	// the rule it reports an unreadable representation under. Both are empty
	// for the string kind, whose New already takes text.
	FromString, ParseRule string
	// Examples are the literals of the directive's own example= tokens, and
	// Samples those followed by the literals the rules of the directive
	// declare valid, each for itself: the constructor decides which of them
	// the whole directive accepts, when the test runs.
	Examples, Samples []string
	// SQL marks a value object with the database/sql codec, and Schema one
	// with a JSONSchema method.
	SQL    bool
	Schema *schemaView
	// Cases are the rejected rows.
	Cases []caseView
	// Normalizations are the rewrites the normalizers of the directive declare.
	Normalizations []normCaseView

	// Catalogue, Members and Field describe an enum; Field is also the name
	// every value object reports its failures under.
	Catalogue string
	Members   []memberView

	// Version is the UUID version an id value object mints.
	Version int
}

// testFile renders and formats the test of one generated file.
func (g *Generator) testFile(file parse.File) ([]byte, error) {
	var bodies bytes.Buffer
	for _, directive := range file.Directives {
		view, name, err := g.newTestView(directive)
		if err != nil {
			return nil, err
		}
		if err := g.tmpl.ExecuteTemplate(&bodies, name+".tmpl", view); err != nil {
			return nil, fmt.Errorf("gen: rendering the test of %s: %w", directive.Name, err)
		}
	}

	imports, err := testImports(bodies.Bytes())
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	std, ext := imports.groups()
	header := fileView{Package: g.opts.Package.Name, Std: std, Ext: ext}
	if err := g.tmpl.ExecuteTemplate(&buf, "header.tmpl", header); err != nil {
		return nil, fmt.Errorf("gen: rendering the test header of %s: %w", file.Path, err)
	}
	buf.Write(bodies.Bytes())

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("gen: formatting the test of %s: %w\n%s", file.Path, err, buf.String())
	}
	return formatted, nil
}

// testImports returns the imports the rendered test bodies reference: testing
// always, and the standard-library packages the checks of a schema use.
func testImports(body []byte) (*importSet, error) {
	imports := newImportSet("")
	for _, want := range []struct{ selector, path string }{
		{"testing.", importTesting},
		{"errors.", importErrors},
		{"regexp.", importRegexp},
		{"slices.", importSlices},
		{"utf8.", importUTF8},
	} {
		if !bytes.Contains(body, []byte(want.selector)) {
			continue
		}
		if err := addAll(imports, want.path); err != nil {
			return nil, err
		}
	}
	return imports, nil
}

// newTestView builds the test data of one directive and names the template
// that renders it.
func (g *Generator) newTestView(d parse.Directive) (testView, string, error) {
	v := testView{Name: d.Name, Field: d.Field, Ctor: "New" + d.Name, SQL: g.opts.SQL}
	if g.opts.Schema {
		v.Schema = newSchemaView(d)
	}

	switch d.Kind {
	case vogue.String:
		v.InType, v.Verb, v.Read = "string", "%q", "String"
		fillScalar(&v, d)
		return v, "test_scalar", nil

	case vogue.Int:
		v.InType, v.Verb, v.Read = "int64", "%v", "Int64"
		v.FromString, v.ParseRule = "New"+d.Name+"FromString", "int"
		fillScalar(&v, d)
		return v, "test_scalar", nil

	case vogue.Decimal:
		v.InType, v.Verb, v.Ctor = "string", "%q", "New"+d.Name+"FromString"
		v.FromString, v.ParseRule = "New"+d.Name+"FromString", "decimal"
		fillScalar(&v, d)
		// A decimal row is written as text and read back as text, and the
		// two need not match: "+1" and "1" are the same number, and only the
		// second is what String reports.
		v.Read = ""
		return v, "test_scalar", nil

	case vogue.Enum:
		for _, member := range d.Values {
			v.Members = append(v.Members, memberView{Method: member.Method, Value: member.Value})
		}
		v.Catalogue = d.Catalogue
		return v, "test_enum", nil

	case vogue.ID:
		if d.Strategy == parse.IDInt64 {
			return v, "test_id_int64", nil
		}
		v.Version = 7
		if d.Strategy == parse.IDUUIDv4 {
			v.Version = 4
		}
		return v, "test_id_uuid", nil

	default:
		return testView{}, "", fmt.Errorf("gen: %s: unsupported kind %s", d.Pos, d.Kind)
	}
}

// fillScalar derives the table of a string, int or decimal constructor from
// the examples the directive and its rules declare.
//
// Which of the rule-declared valid examples the whole directive accepts cannot
// be known here: a rule declares its examples for itself, and `len=2` says
// nothing about a value `required` declared valid. They are therefore handed
// to the suite as candidates, and the suite asks the constructor. Only the
// directive's own examples are asserted outright.
func fillScalar(v *testView, d parse.Directive) {
	rejected := rejections(d)
	for _, use := range d.Rules {
		if use.Rule.Normalize {
			v.Read = ""
			v.Normalizations = append(v.Normalizations, normalizations(d, use)...)
		}
	}

	for _, example := range d.Examples {
		if lit, ok := literal(d.Kind, example); ok {
			v.Examples = append(v.Examples, lit)
		}
	}
	v.Samples = slices.Concat(v.Examples, candidates(d, rejected))
	for _, row := range rejected {
		v.Cases = append(v.Cases, caseView{Note: row.name, Lit: row.lit, Rules: row.rules})
	}
}

// rejection is one input the rules of a directive reject, together with every
// rule that rejects it. Inputs rejected by several rules become a single row,
// which is how the generated table covers error accumulation without any
// example being invented.
type rejection struct {
	name, lit, raw string
	rules          []string
	note           string
}

// rejections collects the rejected examples of a directive in the order the
// rules are written, merging the rows that share an input. Only the rules no
// normalizer precedes contribute.
func rejections(d parse.Directive) []rejection {
	var rows []rejection
	index := map[string]int{}

	normalized := false
	for _, use := range d.Rules {
		if use.Rule.Normalize {
			normalized = true
			continue
		}
		// A rule's invalid example is invalid as the rule sees it. A
		// normalizer written before the rule rewrites the input first — lower
		// turns "EUR" into an accepted "eur" — so the example says nothing
		// about what the constructor does with it, and no row is derived.
		if normalized {
			continue
		}
		for _, example := range use.Rule.Examples.Invalid {
			if !example.AppliesTo(d.Kind, use.Param) {
				continue
			}
			lit, ok := literal(d.Kind, example.In)
			if !ok {
				continue
			}
			if at, seen := index[example.In]; seen {
				rows[at].rules = append(rows[at].rules, use.Rule.Name)
				continue
			}
			index[example.In] = len(rows)
			rows = append(rows, rejection{
				lit: lit, raw: example.In, note: example.Note, rules: []string{use.Rule.Name},
			})
		}
	}

	for i := range rows {
		rows[i].rules = untilPrecondition(d, rows[i].rules)
		rows[i].name = rejectionName(rows[i])
	}
	return rows
}

// untilPrecondition drops the rules a failing precondition keeps from running:
// an empty input fails `required` and would fail `len=2` too, but the
// constructor returns after the first, so only the first is reported. rules
// are in directive order.
func untilPrecondition(d parse.Directive, rules []string) []string {
	for i, name := range rules {
		for _, use := range d.Rules {
			if use.Rule.Name == name && use.Rule.Precondition {
				return rules[:i+1]
			}
		}
	}
	return rules
}

// rejectionName names a rejected row after the note its example carries, or
// after the input and the rules that reject it.
func rejectionName(row rejection) string {
	if row.note != "" {
		return "rejects " + row.note
	}
	return fmt.Sprintf("rejects %s (%s)", strconv.Quote(row.raw), strings.Join(row.rules, ", "))
}

// candidates returns the literals of every valid example a check of the
// directive declares for its parameter, in the order the rules are written,
// leaving out the ones another rule of the directive declares invalid and the
// repeats.
func candidates(d parse.Directive, rejected []rejection) []string {
	var out []string
	seen := map[string]struct{}{}
	for _, use := range d.Rules {
		if use.Rule.Normalize {
			continue
		}
		for _, example := range use.Rule.Examples.Valid {
			if !example.AppliesTo(d.Kind, use.Param) || rejectedBy(rejected, example.In) {
				continue
			}
			lit, ok := literal(d.Kind, example.In)
			if !ok {
				continue
			}
			if _, dup := seen[lit]; dup {
				continue
			}
			seen[lit] = struct{}{}
			out = append(out, lit)
		}
	}
	return out
}

// rejectedBy reports whether another rule of the same directive declares the
// candidate invalid, which is what keeps a sample valid for one rule from
// being asserted valid for the whole value object.
func rejectedBy(rejected []rejection, in string) bool {
	for _, row := range rejected {
		if row.raw == in {
			return true
		}
	}
	return false
}

// normalizations converts the rewrites a normalizer declares into the rows the
// generated test asserts, dropping the ones the kind cannot express. Whether
// the rest of the directive accepts the rewritten value is, like the sample,
// asked of the constructor when the test runs.
func normalizations(d parse.Directive, use parse.RuleUse) []normCaseView {
	var out []normCaseView
	for _, rewrite := range use.Rule.Examples.Normalized {
		in, okIn := literal(d.Kind, rewrite.In)
		out2, okOut := literal(d.Kind, rewrite.Out)
		if !okIn || !okOut {
			continue
		}
		name := rewrite.Note
		if name == "" {
			name = fmt.Sprintf("%s rewrites %s to %s", use.Rule.Name, strconv.Quote(rewrite.In), strconv.Quote(rewrite.Out))
		}
		out = append(out, normCaseView{Name: name, In: in, Out: out2})
	}
	return out
}

// literal renders an example as the Go literal the constructor of the kind
// takes, reporting whether the kind can express it at all: an integer value
// object cannot be handed the example of a rule written for strings.
func literal(kind vogue.Kind, in string) (string, bool) {
	switch kind {
	case vogue.Int:
		if _, err := strconv.ParseInt(in, 10, 64); err != nil {
			return "", false
		}
		return in, true
	case vogue.Decimal:
		// A decimal table is written in text, so the literal is the example
		// itself — but only when it is a decimal at all, which is what keeps a
		// string rule's example out of a decimal table.
		if _, err := decimal.Parse(in); err != nil {
			return "", false
		}
		return strconv.Quote(in), true
	default:
		return strconv.Quote(in), true
	}
}
