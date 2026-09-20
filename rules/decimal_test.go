package rules_test

import (
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// decimalCtx is the emit context of a decimal rule used with the given
// parameter.
func decimalCtx(param string) vogue.EmitContext {
	return vogue.EmitContext{Var: "v", Param: param, Field: "rate", Kind: vogue.Decimal}
}

func TestDecimalRules(t *testing.T) {
	t.Parallel()

	t.Run("the bounds apply to the decimal kind and take a number", func(t *testing.T) {
		t.Parallel()

		// Assert
		for _, rule := range []vogue.Rule{rules.Min, rules.Max} {
			assert.True(t, rule.Kinds.Has(vogue.Decimal), "rule %q should apply to decimal", rule.Name)
			assert.Equal(t, vogue.ParamNumber, rule.Param.Type, "rule %q should take a number", rule.Name)
		}
	})

	t.Run("the sign rules apply to the decimal kind", func(t *testing.T) {
		t.Parallel()

		// Assert
		for _, rule := range []vogue.Rule{rules.Positive, rules.NonNeg} {
			assert.True(t, rule.Kinds.Has(vogue.Decimal), "rule %q should apply to decimal", rule.Name)
		}
	})

	t.Run("oneof stays away from the decimal kind", func(t *testing.T) {
		t.Parallel()

		// Assert
		assert.False(t, rules.OneOf.Kinds.Has(vogue.Decimal),
			"exact equality against a written list is not a question a scaled decimal answers")
	})

	t.Run("a bound compares against a once-parsed declaration", func(t *testing.T) {
		t.Parallel()

		// Act
		expr := rules.Min.Emit(decimalCtx("0.5"))
		decl := rules.Min.Declare(decimalCtx("0.5"))

		// Assert
		assert.Contains(t, expr, "v.Cmp(")
		assert.Contains(t, expr, ") >= 0")
		assert.Contains(t, decl, `decimal.MustParse("0.5")`)
		assert.Contains(t, expr, declaredVar(t, decl))
	})

	t.Run("a bound declares nothing on the kinds that compare without one", func(t *testing.T) {
		t.Parallel()

		// Act
		got := rules.Max.Declare(vogue.EmitContext{Var: "v", Param: "4", Field: "title", Kind: vogue.String})

		// Assert
		assert.Empty(t, got)
	})

	t.Run("the sign rules read the sign of the decimal", func(t *testing.T) {
		t.Parallel()

		// Act & Assert
		assert.Equal(t, "v.Sign() > 0", rules.Positive.Emit(decimalCtx("")))
		assert.Equal(t, "v.Sign() >= 0", rules.NonNeg.Emit(decimalCtx("")))
		assert.Equal(t, "v > 0", rules.Positive.Emit(vogue.EmitContext{Var: "v", Kind: vogue.Int}))
	})

	t.Run("scale bounds the decimal places and rounds nothing", func(t *testing.T) {
		t.Parallel()

		// Assert
		require.NoError(t, rules.Scale.Validate())
		assert.Equal(t, vogue.Kinds(vogue.Decimal), rules.Scale.Kinds)
		assert.Equal(t, vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt}, rules.Scale.Param)
		assert.Equal(t, "v.Scale() <= 4", rules.Scale.Emit(decimalCtx("4")))
		assert.Contains(t, rules.Scale.Doc, "does not round")
	})

	t.Run("nonzero rejects zero at any scale", func(t *testing.T) {
		t.Parallel()

		// Assert
		require.NoError(t, rules.NonZero.Validate())
		assert.Equal(t, vogue.Kinds(vogue.Decimal), rules.NonZero.Kinds)
		assert.Equal(t, "!v.IsZero()", rules.NonZero.Emit(decimalCtx("")))
	})

	t.Run("the new rules are catalogued", func(t *testing.T) {
		t.Parallel()

		// Arrange
		set := rules.MustSet()

		// Act & Assert
		for _, name := range []string{"scale", "nonzero"} {
			_, ok := set.Get(name)
			assert.True(t, ok, "rule %q is missing from the catalogue", name)
		}
	})

	t.Run("every rule that applies to decimal declares a decimal example", func(t *testing.T) {
		t.Parallel()

		// Assert
		for _, rule := range rules.All() {
			if !rule.Kinds.Has(vogue.Decimal) {
				continue
			}
			assert.True(t, hasDecimalExample(rule),
				"rule %q applies to decimal but declares no example written for it", rule.Name)
		}
	})
}

// declaredVar returns the variable name a `var <name> = ...` declaration
// introduces.
func declaredVar(t *testing.T, decl string) string {
	t.Helper()

	fields := strings.Fields(decl)
	require.GreaterOrEqual(t, len(fields), 2, "not a var declaration: %q", decl)
	return fields[1]
}

// hasDecimalExample reports whether a rule declares at least one valid and one
// invalid example usable on the decimal kind.
func hasDecimalExample(rule vogue.Rule) bool {
	usable := func(examples []vogue.Example) bool {
		for _, e := range examples {
			if e.Kinds.Empty() || e.Kinds.Has(vogue.Decimal) {
				return true
			}
		}
		return false
	}
	return usable(rule.Examples.Valid) && usable(rule.Examples.Invalid)
}
