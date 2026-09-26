package voguetest

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/validation"
)

// Enum is the suite of an enum value object.
type Enum[V ValueObject[V], P Pointer[V]] struct {
	// Field is the name the enum reports its failures under.
	Field string
	// All is what the catalogue's All returns.
	All []V
	// Parse is the catalogue's Parse.
	Parse func(string) (V, error)
	// Want are the wire values of the members, in declaration order. Pinning
	// them here is what makes a member added to or removed from the directive
	// fail the test first.
	Want []string
}

// Run runs the suite.
func (s Enum[V, P]) Run(t *testing.T) {
	t.Helper()

	t.Run("lists every member in declaration order", func(t *testing.T) {
		t.Parallel()

		require.Len(t, s.All, len(s.Want))

		for i, member := range s.All {
			assert.Equal(t, s.Want[i], member.String())
			assert.False(t, member.IsZero())
		}
	})

	t.Run("parses every member", func(t *testing.T) {
		t.Parallel()

		for i, raw := range s.Want {
			got, err := s.Parse(raw)
			require.NoError(t, err)
			assert.True(t, s.All[i].Equal(got))
		}
	})

	t.Run("rejects what is not a member", func(t *testing.T) {
		t.Parallel()

		for _, raw := range []string{"not-a-member", ""} {
			got, err := s.Parse(raw)
			require.ErrorIs(t, err, validation.FieldError{Field: s.Field, Rule: "oneof"})
			assert.True(t, got.IsZero())
		}
	})

	t.Run("round trips every member", func(t *testing.T) {
		t.Parallel()

		for _, member := range s.All {
			roundTrips[V, P](t, member)
		}
	})

	if hasScan[V, P]() {
		t.Run("scans", func(t *testing.T) {
			t.Parallel()

			var got V

			scan, _ := scanner[V, P](&got)

			require.NoError(t, scan.Scan(nil))
			assert.True(t, got.IsZero(), "a NULL column must produce the zero value")
			require.ErrorIs(t, scan.Scan("not-a-member"), validation.ErrInvalid)
			require.ErrorIs(t, scan.Scan(struct{}{}), validation.ErrUnsupportedSource)
		})
	}

	t.Run("keeps the zero value out of the members", func(t *testing.T) {
		t.Parallel()

		var zero V

		assert.True(t, zero.IsZero())
		assert.Empty(t, zero.String())

		for _, member := range s.All {
			assert.False(t, zero.Equal(member), "the zero value must not equal %s", member.String())
		}
	})
}
