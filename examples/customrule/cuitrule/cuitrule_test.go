package cuitrule_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue/examples/customrule/cuit"
	"github.com/MathiasHilgert/vogue/examples/customrule/cuitrule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRule(t *testing.T) {
	t.Run("is a rule the generator accepts", func(t *testing.T) {
		// Act
		err := cuitrule.Rule.Validate()

		// Assert
		require.NoError(t, err)
	})

	t.Run("every example agrees with the predicate", func(t *testing.T) {
		// Assert
		for _, example := range cuitrule.Rule.Examples.Valid {
			assert.True(t, cuit.Valid(example.In), "the valid example %q is rejected", example.In)
		}
		for _, example := range cuitrule.Rule.Examples.Invalid {
			assert.False(t, cuit.Valid(example.In), "the invalid example %q is accepted", example.In)
		}
	})
}
