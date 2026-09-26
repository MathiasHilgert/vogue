package gen_test

import (
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

func TestTestGenerator_Candidates(t *testing.T) {
	t.Run("hands every declared valid example to the suite, the zero value included", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(zeroValidRule("0", "7")))
		pkg := parseSource(t, "//vogue:int Stock nonneg\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 2)
		assert.Regexp(t, `Candidates:\s+\[\]int64\{0, 7\},`, string(files[1].Content),
			"the suite, not the generator, skips a zero sample when the test runs")
	})

	t.Run("puts the directive's own examples ahead of the candidates", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(zeroValidRule("0")))
		pkg := parseSource(t, "//vogue:int Stock nonneg example=12 example=40\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		test := string(files[1].Content)
		assert.Regexp(t, `Examples:\s+\[\]int64\{12, 40\},`, test)
		assert.Regexp(t, `Candidates:\s+\[\]int64\{0\},`, test)
	})
}

// normalizerOnlyRule is a normalizer with no companion check, which is the
// shape that leaves the constructor table without an accepted sample: nothing
// in the directive declares a valid input, yet the rewrite it performs is
// still a promise worth testing.
func normalizerOnlyRule() vogue.Rule {
	return vogue.Rule{
		Name:      "lower",
		Kinds:     vogue.Kinds(vogue.String),
		Doc:       "Folds the value to lower case.",
		Message:   "{{.Field}} is lower case",
		Normalize: true,
		Imports:   []string{"strings"},
		Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.ToLower(" + c.Var + ")" },
		Examples: vogue.Examples{
			Normalized: []vogue.Normalization{
				{In: "ÁBC", Out: "ábc", Note: "lower folds an accented capital"},
			},
		},
	}
}

func TestTestGenerator_NormalizerOnlyDirective(t *testing.T) {
	t.Run("proves the rewrite even when no rule declares an accepted sample", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(normalizerOnlyRule()))
		pkg := parseSource(t, "//vogue:string Code lower\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 2)
		test := string(files[1].Content)
		assert.Regexp(t, `Candidates:\s+nil,`, test)
		assert.Contains(t, test, `Out:   "ábc",`)
	})

	t.Run("imports only testing and the suites", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(normalizerOnlyRule()))
		pkg := parseSource(t, "//vogue:string Code lower\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		test := string(files[1].Content)
		assert.Contains(t, test, "import (\n\t\"testing\"\n\n\t\"github.com/MathiasHilgert/vogue/voguetest\"\n)")
	})
}

// trimRule is the normalizer whose rewrites the tests below argue about.
func trimRule() vogue.Rule {
	return vogue.Rule{
		Name:      "trim",
		Kinds:     vogue.Kinds(vogue.String),
		Doc:       "Removes surrounding whitespace.",
		Message:   "{{.Field}} is trimmed",
		Normalize: true,
		Imports:   []string{"strings"},
		Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.TrimSpace(" + c.Var + ")" },
		Examples: vogue.Examples{
			Normalized: []vogue.Normalization{{In: "  Tortilla  ", Out: "Tortilla", Note: "the blanks are dropped"}},
		},
	}
}

// narrowRule stands for a check whose accepted set is small and specific — a
// checksum, a code list — which is the case where a normalizer's own example
// says nothing about whether the directive as a whole accepts the rewrite.
func narrowRule() vogue.Rule {
	return vogue.Rule{
		Name:    "cuit",
		Kinds:   vogue.Kinds(vogue.String),
		Doc:     "Requires a checksummed identifier.",
		Message: "{{.Field}} must be a valid identifier",
		Emit:    func(c vogue.EmitContext) string { return "len(" + c.Var + ") == 11" },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{In: "20123456786", Note: "a well-formed identifier"}},
			Invalid: []vogue.Example{{In: "123", Note: "too short"}},
		},
	}
}

// broadRule stands for a check that declares the normalizer's own output
// acceptable, which is what makes the rewrite provable.
func broadRule() vogue.Rule {
	return vogue.Rule{
		Name:    "required",
		Kinds:   vogue.Kinds(vogue.String),
		Doc:     "Rejects the empty value.",
		Message: "{{.Field}} is required",
		Emit:    func(c vogue.EmitContext) string { return c.Var + ` != ""` },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{In: "Tortilla", Note: "an ordinary value"}},
			Invalid: []vogue.Example{{In: "", Note: "the empty string"}},
		},
	}
}

func TestTestGenerator_Normalizations(t *testing.T) {
	t.Run("hands a rewrite to the suite even when no rule vouches for its output", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(trimRule(), narrowRule()))
		pkg := parseSource(t, "//vogue:string TaxID trim cuit\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		assert.Contains(t, string(files[1].Content), `Out:   "Tortilla",`,
			"the suite asks the constructor whether the rest of the directive accepts the rewrite")
	})

	t.Run("keeps a rewrite the checks of the directive declare acceptable", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(trimRule(), broadRule()))
		pkg := parseSource(t, "//vogue:string Name trim required\n", set)

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, Rules: set})

		// Assert
		require.NoError(t, err)
		test := string(files[1].Content)
		assert.Contains(t, test, "Normalized: []voguetest.Normalization[string]{")
		assert.Contains(t, test, `Name:  "the blanks are dropped",`)
	})
}
