package cuit_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue/examples/customrule/cuit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValid(t *testing.T) {
	t.Parallel()

	// Arrange
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "accepts an individual CUIT", in: "20123456786", want: true},
		{name: "accepts a check digit that carries to zero", in: "27123456780", want: true},
		{name: "rejects the wrong check digit", in: "20123456789", want: false},
		{name: "rejects ten digits", in: "2012345678", want: false},
		{name: "rejects twelve digits", in: "201234567866", want: false},
		{name: "rejects a punctuated CUIT", in: "20-12345678-6", want: false},
		{name: "rejects a non-digit in the check position", in: "2012345678X", want: false},
		{name: "rejects a non-digit inside the body", in: "2012345X786", want: false},
		{name: "rejects the empty string", in: "", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got := cuit.Valid(tc.in)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRule(t *testing.T) {
	t.Run("is a rule the generator accepts", func(t *testing.T) {
		// Act
		err := cuit.Rule.Validate()

		// Assert
		require.NoError(t, err)
	})

	t.Run("every example agrees with the predicate", func(t *testing.T) {
		// Assert
		for _, example := range cuit.Rule.Examples.Valid {
			assert.True(t, cuit.Valid(example.In), "the valid example %q is rejected", example.In)
		}
		for _, example := range cuit.Rule.Examples.Invalid {
			assert.False(t, cuit.Valid(example.In), "the invalid example %q is accepted", example.In)
		}
	})
}
