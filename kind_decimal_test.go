package vogue_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecimalKind(t *testing.T) {
	t.Parallel()

	t.Run("the decimal kind has a directive spelling", func(t *testing.T) {
		t.Parallel()

		// Act
		got := vogue.Decimal.String()

		// Assert
		assert.Equal(t, "decimal", got)
	})

	t.Run("ParseKind resolves the decimal spelling", func(t *testing.T) {
		t.Parallel()

		// Act
		got, err := vogue.ParseKind("decimal")

		// Assert
		require.NoError(t, err)
		assert.Equal(t, vogue.Decimal, got)
	})

	t.Run("decimal is a member of a KindSet that declares it", func(t *testing.T) {
		t.Parallel()

		// Arrange
		set := vogue.Kinds(vogue.Decimal, vogue.Int)

		// Act & Assert
		assert.True(t, set.Has(vogue.Decimal))
		assert.False(t, set.Has(vogue.String))
		assert.Equal(t, "int, decimal", set.String())
	})

	t.Run("decimal sorts after the kinds that shipped before it", func(t *testing.T) {
		t.Parallel()

		// Act
		got := vogue.Kinds(vogue.Decimal, vogue.String, vogue.Int).Kinds()

		// Assert
		assert.Equal(t, []vogue.Kind{vogue.String, vogue.Int, vogue.Decimal}, got)
	})
}

func TestParamNumber(t *testing.T) {
	t.Parallel()

	t.Run("the number parameter type has a diagnostic name", func(t *testing.T) {
		t.Parallel()

		// Act
		got := vogue.ParamNumber.String()

		// Assert
		assert.Equal(t, "number", got)
	})

	t.Run("a rule taking a number parameter validates", func(t *testing.T) {
		t.Parallel()

		// Arrange
		rule := vogue.Rule{
			Name:    "bound",
			Kinds:   vogue.Kinds(vogue.Decimal),
			Message: "{{.Field}} is out of bounds",
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamNumber},
			Emit:    func(c vogue.EmitContext) string { return c.Var + ".Sign() > 0" },
		}

		// Act
		err := rule.Validate()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "required number", rule.Param.String())
	})
}
