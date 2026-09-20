package gen_test

import (
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// zeroValidRule declares a valid example that happens to be the zero value of
// its kind, which is what `nonneg` does: zero is a perfectly good stock level.
// A generated test cannot assert that such a sample produced a non-zero value
// object, so the sample has to be chosen with that in mind.
func zeroValidRule(valid ...string) vogue.Rule {
	examples := make([]vogue.Example, len(valid))
	for i, in := range valid {
		examples[i] = vogue.Example{In: in, Note: "accepted " + in}
	}
	return vogue.Rule{
		Name:    "nonneg",
		Kinds:   vogue.Kinds(vogue.Int),
		Doc:     "Rejects values below zero.",
		Message: "{{.Field}} must not be negative",
		Emit:    func(c vogue.EmitContext) string { return c.Var + " >= 0" },
		Examples: vogue.Examples{
			Valid:   examples,
			Invalid: []vogue.Example{{In: "-1", Note: "one below the floor"}},
		},
	}
}

func TestTestGenerator_AcceptedSample(t *testing.T) {
	t.Run("skips the zero value when another accepted example is declared", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(zeroValidRule("0", "7")))
		pkg := parseSource(t, "//vogue:int Stock nonneg\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 2)
		test := string(files[1].Content)
		assert.Contains(t, test, `{name: "accepts \"7\"", in: 7}`)
		assert.NotContains(t, test, `{name: "accepts \"0\""`)
		assert.NotContains(t, test, "t.Skip")
	})

	t.Run("skips the test rather than asserting a zero sample is not zero", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(zeroValidRule("0")))
		pkg := parseSource(t, "//vogue:int Stock nonneg\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		test := string(files[1].Content)
		assert.Contains(t, test, "t.Skip")
		assert.True(t, strings.Contains(test, "example"), "the skip must ask for an example")
	})
}
