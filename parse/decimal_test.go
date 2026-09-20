package parse_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decimalRules is the catalogue the decimal parser tests run against: one
// number-parameter rule spanning int and decimal, and one decimal-only rule,
// which is exactly the shape the built-in catalogue grew in T7.
func decimalRules(t *testing.T) *vogue.RuleSet {
	t.Helper()

	always := func(vogue.EmitContext) string { return "true" }
	set := &vogue.RuleSet{}
	require.NoError(t, set.Add(
		vogue.Rule{
			Name: "min", Kinds: vogue.Kinds(vogue.String, vogue.Int, vogue.Decimal),
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamNumber},
			Message: "{{.Field}} must be at least {{.Param}}", Emit: always,
		},
		vogue.Rule{
			Name: "scale", Kinds: vogue.Kinds(vogue.Decimal),
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
			Message: "{{.Field}} must have at most {{.Param}} decimal places", Emit: always,
		},
		vogue.Rule{
			Name: "positive", Kinds: vogue.Kinds(vogue.Int, vogue.Decimal),
			Message: "{{.Field}} must be greater than zero", Emit: always,
		},
	))
	return set
}

// parseDecimalSrc runs the parser over one in-memory file using [decimalRules].
func parseDecimalSrc(t *testing.T, src string) (*parse.Package, error) {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "vo.go", src, parser.ParseComments)
	require.NoError(t, err)
	return parse.Files(fset, []*ast.File{file}, decimalRules(t))
}

func TestFilesDecimal(t *testing.T) {
	t.Parallel()

	t.Run("a decimal directive carries its rules in the order written", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := `package tab

// DiscountRate is the share of the bill that is taken off.
//vogue:decimal DiscountRate min=0.5 scale=4
`

		// Act
		pkg, err := parseDecimalSrc(t, src)

		// Assert
		require.NoError(t, err)
		require.Len(t, pkg.Files[0].Directives, 1)
		d := pkg.Files[0].Directives[0]
		assert.Equal(t, vogue.Decimal, d.Kind)
		assert.Equal(t, "DiscountRate", d.Name)
		assert.Equal(t, "discountRate", d.Field)
		assert.Equal(t, []string{"min=0.5", "scale=4"}, ruleUses(d))
	})

	t.Run("a fractional bound is rejected on the int kind", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package tab\n\n//vogue:int Covers min=0.5\n"

		// Act
		_, err := parseDecimalSrc(t, src)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `invalid parameter for rule "min": "0.5" is not an integer`)
	})

	t.Run("a bound that is not a number at all is rejected on the decimal kind", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package tab\n\n//vogue:decimal Rate min=half\n"

		// Act
		_, err := parseDecimalSrc(t, src)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `invalid parameter for rule "min": "half" is not a decimal number`)
	})

	t.Run("a bound wider than a decimal can hold is rejected", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package tab\n\n//vogue:decimal Rate min=99999999999999999999999\n"

		// Act
		_, err := parseDecimalSrc(t, src)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `invalid parameter for rule "min": "99999999999999999999999" is not a decimal number`)
	})

	t.Run("a bound the library would silently round is rejected", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package tab\n\n//vogue:decimal Rate min=0.12345678901234567890123\n"

		// Act
		_, err := parseDecimalSrc(t, src)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(),
			`invalid parameter for rule "min": "0.12345678901234567890123" has more than 19 decimal places`)
	})

	t.Run("an exponent bound is accepted", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package tab\n\n//vogue:decimal Rate min=1.5e2\n"

		// Act
		pkg, err := parseDecimalSrc(t, src)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"min=1.5e2"}, ruleUses(pkg.Files[0].Directives[0]))
	})

	t.Run("a decimal-only rule does not apply to the int kind", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package tab\n\n//vogue:int Covers scale=2\n"

		// Act
		_, err := parseDecimalSrc(t, src)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `rule "scale" does not apply to kind int`)
		assert.Contains(t, err.Error(), "it applies to: decimal")
	})

	t.Run("a misspelled decimal kind is suggested", func(t *testing.T) {
		t.Parallel()

		// Arrange
		src := "package tab\n\n//vogue:decimel Rate positive\n"

		// Act
		_, err := parseDecimalSrc(t, src)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown kind "decimel"`)
		assert.Contains(t, err.Error(), `did you mean "decimal"?`)
	})
}
