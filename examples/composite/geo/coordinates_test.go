package geo_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/examples/composite/geo"
	"github.com/MathiasHilgert/vogue/validation"
)

func TestNewCoordinates(t *testing.T) {
	t.Parallel()

	t.Run("builds a point from two valid parts", func(t *testing.T) {
		t.Parallel()

		// Act
		got, err := geo.NewCoordinates("-34.603722", "-58.381592")

		// Assert
		require.NoError(t, err)
		assert.False(t, got.IsZero())
		assert.Equal(t, "-34.603722", got.Latitude().String())
		assert.Equal(t, "-58.381592", got.Longitude().String())
		assert.Equal(t, "-34.603722,-58.381592", got.String())
	})

	t.Run("reports the failures of both parts at once", func(t *testing.T) {
		t.Parallel()

		// Act
		got, err := geo.NewCoordinates("91", "-181")

		// Assert
		require.ErrorIs(t, err, validation.ErrInvalid)
		require.ErrorIs(t, err, validation.Failure("latitude", "max"))
		require.ErrorIs(t, err, validation.Failure("longitude", "min"))
		assert.True(t, got.IsZero())
	})

	t.Run("round trips through its text form", func(t *testing.T) {
		t.Parallel()

		// Arrange
		want, err := geo.NewCoordinates("40.416775", "-3.70379")
		require.NoError(t, err)

		// Act
		text, err := want.MarshalText()
		require.NoError(t, err)

		var got geo.Coordinates
		err = got.UnmarshalText(text)

		// Assert
		require.NoError(t, err)
		assert.True(t, want.Equal(got))
	})

	t.Run("refuses text that is not a pair", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var got geo.Coordinates

		// Act
		err := got.UnmarshalText([]byte("40.4"))

		// Assert
		require.ErrorIs(t, err, validation.Failure("coordinates", "pair"))
	})
}
