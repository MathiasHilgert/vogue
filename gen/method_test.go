package gen_test

import (
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// methodRule is a rule whose check lives in an unexported method of the
// generated type, the shape the email, url, uuid and timezone rules have.
func methodRule(name, body string, kinds vogue.KindSet) vogue.Rule {
	return vogue.Rule{
		Name:    "method",
		Kinds:   kinds,
		Doc:     "Rejects values the method rejects.",
		Message: "{{.Field}} is not acceptable",
		Imports: []string{"strings"},
		Method:  func(vogue.EmitContext) (string, string) { return name, body },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{In: "a", Kinds: vogue.Kinds(vogue.String)}},
			Invalid: []vogue.Example{{In: "b", Kinds: vogue.Kinds(vogue.String)}},
		},
	}
}

func methodSet(t *testing.T, rule vogue.Rule) *vogue.RuleSet {
	t.Helper()

	set := &vogue.RuleSet{}
	require.NoError(t, set.Add(rule))
	return set
}

func TestGenerator_RuleMethods(t *testing.T) {
	const body = "return strings.Contains(value, \"a\")"

	t.Run("writes the method on the type and calls it from the constructor", func(t *testing.T) {
		// Arrange
		set := methodSet(t, methodRule("containsLetterA", body, vogue.Kinds(vogue.String)))
		pkg := parseSource(t, "//vogue:string Code method\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		code := string(files[0].Content)
		assert.Contains(t, code, "func (Code) containsLetterA(value string) bool {")
		assert.Contains(t, code, "\treturn strings.Contains(value, \"a\")\n}")
		assert.Contains(t, code, "if !code.containsLetterA(value) {")
		assert.Contains(t, code, "var code Code // stays zero unless every rule passes")
		assert.Contains(t, code, "\t\"strings\"\n")
		assert.Less(t, strings.Index(code, "func (code Code) MarshalText"), strings.Index(code, "func (Code) containsLetterA"),
			"exported methods come before unexported ones")
	})

	t.Run("writes one method per type, each on its own type", func(t *testing.T) {
		// Arrange
		set := methodSet(t, methodRule("containsLetterA", body, vogue.Kinds(vogue.String)))
		pkg := parseSource(t, "//vogue:string Code method\n\n//vogue:string Other method\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		code := string(files[0].Content)
		assert.Equal(t, 1, strings.Count(code, "func (Code) containsLetterA("))
		assert.Equal(t, 1, strings.Count(code, "func (Other) containsLetterA("))
	})

	t.Run("types the parameter after the kind", func(t *testing.T) {
		// Arrange
		set := methodSet(t, methodRule("isEven", "return value%2 == 0", vogue.Kinds(vogue.Int)))
		pkg := parseSource(t, "//vogue:int Count method\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(files[0].Content), "func (Count) isEven(value int64) bool {")
	})

	t.Run("types the parameter of a decimal as the decimal type", func(t *testing.T) {
		// Arrange
		set := methodSet(t, methodRule("isPositive", "return value.Sign() > 0", vogue.Kinds(vogue.Decimal)))
		pkg := parseSource(t, "//vogue:decimal Price method\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(files[0].Content), "func (Price) isPositive(value decimal.Decimal) bool {")
	})

	for name, method := range map[string]string{
		"an exported name":  "ContainsLetterA",
		"an empty name":     "",
		"a keyword":         "func",
		"a non-identifier":  "contains-a",
		"a generated name":  "value",
		"an existing field": "set",
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			// Arrange
			set := methodSet(t, methodRule(method, body, vogue.Kinds(vogue.String)))
			pkg := parseSource(t, "//vogue:string Code method\n", set)

			// Act
			_, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Rules: set})

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "method")
		})
	}

	t.Run("marks a method longer than funlen's default limit with a targeted directive", func(t *testing.T) {
		// Arrange
		long := strings.Repeat("\t_ = value\n", 70) + "\treturn true"
		set := methodSet(t, methodRule("isListed", long, vogue.Kinds(vogue.String)))
		pkg := parseSource(t, "//vogue:string Code method\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(files[0].Content), "//nolint:funlen // ")
	})

	t.Run("leaves a short method without a directive", func(t *testing.T) {
		// Arrange
		set := methodSet(t, methodRule("isListed", body, vogue.Kinds(vogue.String)))
		pkg := parseSource(t, "//vogue:string Code method\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		assert.NotContains(t, string(files[0].Content), "nolint:funlen")
	})
}
