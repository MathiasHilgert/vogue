package gen_test

import (
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/gen/internal/testrules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateDecimal renders one decimal directive and returns the value-object
// file and its test.
func generateDecimal(t *testing.T, directive string) (code, test string) {
	t.Helper()

	rules := testrules.Set("example.com/tab")
	pkg := parseSource(t, directive, rules)
	files, err := filesOf(t, gen.Options{Package: pkg, Rules: rules})
	require.NoError(t, err)
	require.Len(t, files, 2)
	require.Equal(t, gen.CodeFile, files[0].Kind)
	require.Equal(t, gen.TestFile, files[1].Kind)
	return string(files[0].Content), string(files[1].Content)
}

// countOf returns how many times needle occurs in s.
func countOf(s, needle string) int { return strings.Count(s, needle) }

func TestGenerateDecimal(t *testing.T) {
	t.Parallel()

	t.Run("the generated type wraps a decimal and exposes it", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:decimal Weight min=0 scale=3\n")

		// Assert
		assert.Contains(t, code, `"github.com/govalues/decimal"`)
		assert.Contains(t, code, "type Weight struct{ v decimal.Decimal }")
		assert.Contains(t, code, "func NewWeight(raw decimal.Decimal) (Weight, error)")
		assert.Contains(t, code, "func ParseWeight(raw string) (Weight, error)")
		assert.Contains(t, code, "func (w Weight) Decimal() decimal.Decimal")
		assert.Contains(t, code, "func (w Weight) String() string")
		assert.Contains(t, code, "func (w Weight) IsZero() bool")
		assert.Contains(t, code, "func (w Weight) MarshalText() ([]byte, error)")
		assert.Contains(t, code, "func (w *Weight) UnmarshalText(data []byte) error")
	})

	t.Run("equality compares the numbers, not their representations", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:decimal Weight min=0\n")

		// Assert
		assert.Contains(t, code, "func (w Weight) Equal(other Weight) bool { return w.v.Cmp(other.v) == 0 }")
	})

	t.Run("Value stores the canonical text and Scan refuses binary floats", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:decimal Weight min=0\n")

		// Assert
		assert.Contains(t, code, "func (w Weight) Value() (driver.Value, error) { return w.v.String(), nil }")
		assert.Contains(t, code, "case float64, float32:")
		assert.Contains(t, code, "cannot scan the binary float %T into Weight")
	})

	t.Run("a bound is parsed once into a package-level declaration", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:decimal Weight min=0.5\n")

		// Assert
		assert.Contains(t, code, `= decimal.MustParse("0.5")`)
		assert.Contains(t, code, ".Cmp(_vogueDecimal")
		assert.NotContains(t, code, "decimal.MustParse(\"0.5\")) >= 0")
	})

	t.Run("a bound shared by two directives is declared once", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:decimal Weight min=0.5\n\n//vogue:decimal Rate min=0.5\n")

		// Assert
		assert.Equal(t, 1, countOf(code, `decimal.MustParse("0.5")`))
	})

	t.Run("a parse failure is reported as a decimal field error", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:decimal Weight min=0\n")

		// Assert
		assert.Contains(t, code, `Rule: "decimal"`)
		assert.Contains(t, code, "weight must be an exact decimal number")
	})

	t.Run("a scale bound compares the scale of the value", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:decimal Weight scale=3\n")

		// Assert
		assert.Contains(t, code, "v.Scale() <= 3")
	})

	t.Run("the generated test drives the constructor through Parse", func(t *testing.T) {
		t.Parallel()

		// Act
		_, test := generateDecimal(t, "//vogue:decimal Weight min=0\n")

		// Assert
		assert.Contains(t, test, "func TestNewWeight(t *testing.T)")
		assert.Contains(t, test, "in        string")
		assert.Contains(t, test, "got, err := ParseWeight(tc.in)")
		assert.Contains(t, test, "a binary float source is refused")
	})

	t.Run("a string directive does not import decimal", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:string Title min=1\n")

		// Assert
		assert.NotContains(t, code, "govalues/decimal")
	})

	t.Run("an int directive imports neither decimal nor utf8", func(t *testing.T) {
		t.Parallel()

		// Act
		code, _ := generateDecimal(t, "//vogue:int Covers min=1\n")

		// Assert
		assert.NotContains(t, code, "govalues/decimal")
		assert.NotContains(t, code, "unicode/utf8")
	})
}
