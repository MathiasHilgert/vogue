package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"strconv"
	"strings"

	"github.com/govalues/decimal"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
)

// Import paths only the generated tests need.
const (
	importTesting = "testing"
	importAssert  = "github.com/stretchr/testify/assert"
	importRequire = "github.com/stretchr/testify/require"
)

// The UUID literals the generated id tests parse. One of each version the
// generator mints, so the test proves the documented promise that Parse reads
// any RFC 4122 UUID and not only the version this type produces.
const (
	sampleUUIDv4 = "3f333df6-90a4-4fda-8dd3-9485d27cee36"
	sampleUUIDv7 = "018ff1d4-9c2a-7b3e-9f6a-6c1d2e3f4a5b"
	zeroUUID     = "00000000-0000-0000-0000-000000000000"
)

// caseView is one row of a generated table-driven test: the input, the rules
// expected to reject it, and the name the subtest runs under. An empty Rules
// means the row must be accepted.
type caseView struct {
	Name  string
	Lit   string
	Rules []string
}

// normCaseView is one rewrite a normalizer performs, as the generated test
// states it: feed In to the constructor and expect the value object to hold
// Out afterwards.
type normCaseView struct {
	Name    string
	In, Out string
}

// testView is the data a test template renders. As with [voView], the fields
// that do not apply to the kind being rendered stay at their zero value.
type testView struct {
	// Name is the value-object type, Recv its receiver, Field the name it
	// reports failures under.
	Name, Recv, Field string

	// Skip carries the reason the constructor cannot be exercised, which is
	// what the generated test skips with. It is empty when a sample was found.
	Skip string
	// InType is the Go type the table feeds [testView.Ctor], and Accessor the
	// method that reads the value back.
	InType, Accessor string
	// Ctor is the constructor the table drives. It is New<Name> for every kind
	// whose constructor takes a Go literal the table can write; the decimal
	// kind takes a decimal.Decimal, which no literal can spell, so its table
	// is written in the text its Parse<Name> reads.
	Ctor string
	// HasParse marks a kind that also exposes Parse<Name>, ParseRule the rule
	// its failures are reported under and ParseNote how the generated subtest
	// names an input it cannot read.
	HasParse             bool
	ParseRule, ParseNote string
	// RefusesFloat marks a kind whose Scan refuses a binary float outright,
	// which is a promise worth a test of its own.
	RefusesFloat bool
	// Cases are the constructor rows, the accepted sample first.
	Cases []caseView
	// ValidLit is the literal of the accepted sample and ValidRaw its textual
	// spelling, used by the round-trip tests.
	ValidLit, ValidRaw string
	// InvalidRaw is the textual spelling of a rejected sample, empty when no
	// rule declared one. It proves Scan re-runs validation.
	InvalidRaw string
	// AssertEqual allows the accepted row to assert the value is held
	// unchanged, which only holds when no normalizer rewrites it.
	AssertEqual bool
	// Normalizations are the rewrites the normalizers of the directive declare.
	Normalizations []normCaseView

	// Members and MemberParam describe an enum.
	Members     []memberView
	MemberParam string

	// Version is the UUID version an id value object mints, Samples the UUID
	// literals its parser is proven against, and Zero the nil UUID an
	// unassigned identifier reads back as.
	Version int
	Samples []string
	Zero    string
}

// testFile renders and formats the test of one generated file.
func (g *Generator) testFile(file parse.File) ([]byte, error) {
	var bodies bytes.Buffer
	for _, directive := range file.Directives {
		view, name, err := newTestView(directive)
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

// testImports returns the imports the rendered test bodies actually reference.
//
// Which helpers a test uses depends on what its directive declared: a
// directive whose rules declare no accepted sample renders a skipped
// constructor test and, when it normalizes, nothing but the rewrite table, so
// neither vogue nor testify is named. Adding the four unconditionally would
// leave such a file with imports the compiler rejects, so the set is derived
// from the body rather than assumed.
func testImports(body []byte) (*importSet, error) {
	imports := newImportSet("")
	wanted := []struct {
		selector string
		path     string
	}{
		{"testing.", importTesting},
		{"vogue.", importVogue},
		{"assert.", importAssert},
		{"require.", importRequire},
	}
	// Every generated test takes *testing.T, so testing is always named; the
	// loop still asks, because that is what keeps the rule one rule.
	for _, want := range wanted {
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
func newTestView(d parse.Directive) (testView, string, error) {
	v := testView{Name: d.Name, Recv: receiver(d.Name), Field: d.Field, Ctor: "New" + d.Name}

	switch d.Kind {
	case vogue.String:
		v.InType, v.Accessor = "string", "String"
		fillScalar(&v, d)
		return v, "test_scalar", nil

	case vogue.Int:
		v.InType, v.Accessor, v.HasParse = "int64", "Int64", true
		v.ParseRule, v.ParseNote = "int", "is not a whole number"
		fillScalar(&v, d)
		return v, "test_scalar", nil

	case vogue.Decimal:
		v.InType, v.Accessor, v.HasParse = "string", "String", true
		v.Ctor, v.ParseRule, v.ParseNote = "Parse"+d.Name, "decimal", "is not an exact decimal number"
		v.RefusesFloat = true
		fillScalar(&v, d)
		return v, "test_scalar", nil

	case vogue.Enum:
		values := make([]string, len(d.Values))
		for i, member := range d.Values {
			v.Members = append(v.Members, memberView{Const: member.Const, Value: member.Value})
			values[i] = member.Value
		}
		v.MemberParam = strings.Join(values, ",")
		return v, "test_enum", nil

	case vogue.ID:
		if d.Strategy == parse.IDInt64 {
			return v, "test_id_int64", nil
		}
		v.Version, v.Samples, v.Zero = 7, []string{sampleUUIDv7, sampleUUIDv4}, zeroUUID
		if d.Strategy == parse.IDUUIDv4 {
			v.Version = 4
		}
		return v, "test_id_uuid", nil

	default:
		return testView{}, "", fmt.Errorf("gen: %s: unsupported kind %s", d.Pos, d.Kind)
	}
}

// fillScalar derives the table of a string or int constructor from the
// examples the rules of the directive declare.
func fillScalar(v *testView, d parse.Directive) {
	// A decimal row is written as text and read back as text, and the two need
	// not match: "+1" and "1" are the same number, and only the second is what
	// String reports. Asserting the value came through unchanged is therefore
	// something only the kinds with one spelling per value can promise.
	v.AssertEqual = d.Kind != vogue.Decimal
	rejected := rejections(d)
	for _, use := range d.Rules {
		if use.Rule.Normalize {
			v.AssertEqual = false
			v.Normalizations = append(v.Normalizations, normalizations(d, use, rejected)...)
		}
	}

	v.ValidLit, v.ValidRaw = acceptedSample(d, rejected)
	if v.ValidLit == "" {
		v.Skip = fmt.Sprintf("vogue: no example of %s satisfies every rule and differs from the zero value; add Examples to the rules it uses", d.Name)
		v.AssertEqual = false
		return
	}

	v.Cases = append(v.Cases, caseView{Name: "accepts " + v.ValidRaw, Lit: v.ValidLit})
	for _, row := range rejected {
		v.Cases = append(v.Cases, caseView{Name: row.name, Lit: row.lit, Rules: row.rules})
		if v.InvalidRaw == "" {
			v.InvalidRaw = row.raw
		}
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
// rules are written, merging the rows that share an input.
func rejections(d parse.Directive) []rejection {
	var rows []rejection
	index := map[string]int{}

	for _, use := range d.Rules {
		if use.Rule.Normalize {
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
		rows[i].name = rejectionName(rows[i])
	}
	return rows
}

// rejectionName names a rejected row after the note its example carries, or
// after the input and the rules that reject it.
func rejectionName(row rejection) string {
	if row.note != "" {
		return "rejects " + row.note
	}
	return fmt.Sprintf("rejects %s (%s)", strconv.Quote(row.raw), strings.Join(row.rules, ", "))
}

// acceptedSample returns the literal and the textual spelling of the first
// declared valid example no rule of the directive rejects and that is not the
// zero value of its kind. Both are empty when the rules declare none.
//
// The zero value is skipped because the generated test asserts that an
// accepted input produced a usable value object, and a value object built from
// the zero value is indistinguishable from one that never passed validation:
// `nonneg` accepting 0 is correct, but "0 is not the zero value" is not
// something any generated code could promise. Rather than emit an assertion
// that must fail, the sample moves on to the next declared example, and the
// test skips when there is none.
func acceptedSample(d parse.Directive, rejected []rejection) (lit, raw string) {
	for _, use := range d.Rules {
		if use.Rule.Normalize {
			continue
		}
		for _, example := range use.Rule.Examples.Valid {
			if !example.AppliesTo(d.Kind, use.Param) {
				continue
			}
			candidate, ok := literal(d.Kind, example.In)
			if !ok || rejectedBy(rejected, example.In) || isZeroLiteral(d.Kind, example.In) {
				continue
			}
			return candidate, strconv.Quote(example.In)
		}
	}
	return "", ""
}

// isZeroLiteral reports whether the example is the zero value of its kind: the
// empty string, or the number zero however it was spelled.
func isZeroLiteral(kind vogue.Kind, in string) bool {
	switch kind {
	case vogue.Int:
		n, err := strconv.ParseInt(in, 10, 64)
		return err == nil && n == 0
	case vogue.Decimal:
		d, err := decimal.Parse(in)
		return err == nil && d.IsZero()
	default:
		return in == ""
	}
}

// accepted reports whether the checks of the directive are known to accept a
// value, which is what a generated assertion may rely on.
//
// A normalizer declares what it rewrites, not what the rest of the directive
// makes of the result: `trim` turning "  Tortilla  " into "Tortilla" says
// nothing about a `cuit` written after it, which rejects both. The evidence
// available at generate time is the declared examples, so a value counts as
// accepted when some check of the directive declares it valid and none
// declares it invalid — the same standard [acceptedSample] applies to the
// constructor table. A directive whose rules only normalize checks nothing, so
// every value passes it.
func accepted(d parse.Directive, rejected []rejection, value string) bool {
	if rejectedBy(rejected, value) {
		return false
	}
	checks := false
	for _, use := range d.Rules {
		if use.Rule.Normalize {
			continue
		}
		checks = true
		for _, example := range use.Rule.Examples.Valid {
			if example.AppliesTo(d.Kind, use.Param) && example.In == value {
				return true
			}
		}
	}
	return !checks
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
// generated test asserts, dropping the ones the kind cannot express and the
// ones the other rules of the directive do not vouch for.
func normalizations(d parse.Directive, use parse.RuleUse, rejected []rejection) []normCaseView {
	var out []normCaseView
	for _, rewrite := range use.Rule.Examples.Normalized {
		in, okIn := literal(d.Kind, rewrite.In)
		out2, okOut := literal(d.Kind, rewrite.Out)
		if !okIn || !okOut || !accepted(d, rejected, rewrite.Out) {
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
