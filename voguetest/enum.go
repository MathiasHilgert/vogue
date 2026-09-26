package voguetest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/validation"
)

// Enum is the suite of an enum value object.
type Enum[Object ValueObject[Object], Reference Pointer[Object]] struct {
	// Field is the name the enum reports its failures under.
	Field string
	// All is what the catalogue's All returns.
	All []Object
	// Parse is the catalogue's Parse.
	Parse func(string) (Object, error)
	// Want are the wire values of the members, in declaration order, which the
	// catalogue must list, parse and nothing else.
	Want []string
}

// Run runs the suite.
func (suite Enum[Object, Reference]) Run(t *testing.T) {
	t.Helper()

	t.Run("lists every member in declaration order", func(t *testing.T) {
		t.Parallel()

		require.Len(t, suite.All, len(suite.Want))

		for index, member := range suite.All {
			assert.Equal(t, suite.Want[index], member.String())
			assert.False(t, member.IsZero())
		}
	})

	t.Run("parses every member", func(t *testing.T) {
		t.Parallel()

		for index, raw := range suite.Want {
			got, err := suite.Parse(raw)
			require.NoError(t, err)
			assert.True(t, suite.All[index].Equal(got))
		}
	})

	t.Run("rejects what is not a member", func(t *testing.T) {
		t.Parallel()

		for _, raw := range []string{"not-a-member", ""} {
			got, err := suite.Parse(raw)
			require.ErrorIs(t, err, validation.FieldError{Field: suite.Field, Rule: "oneof"})
			assert.True(t, got.IsZero())
		}
	})

	t.Run("round trips every member", func(t *testing.T) {
		t.Parallel()

		for _, member := range suite.All {
			roundTrips[Object, Reference](t, member)
			describes(t, member)
		}
	})

	if hasScan[Object, Reference]() {
		t.Run("scans", func(t *testing.T) {
			t.Parallel()

			var got Object

			scan, _ := scanner[Object, Reference](&got)

			require.NoError(t, scan.Scan(nil))
			assert.True(t, got.IsZero(), "a NULL column must produce the zero value")
			require.ErrorIs(t, scan.Scan("not-a-member"), validation.ErrInvalid)
			require.ErrorIs(t, scan.Scan(struct{}{}), validation.ErrUnsupportedSource)
		})
	}

	t.Run("keeps the zero value out of the members", func(t *testing.T) {
		t.Parallel()

		var zero Object

		assert.Empty(t, zero.String())

		for _, member := range suite.All {
			separatesZero(t, member)
		}
	})
}
