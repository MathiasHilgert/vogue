package geo_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/examples/composite/geo"
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
		require.Error(t, err)
		assert.Equal(t, "invalid coordinates: invalid latitude: must be at most 90 (max)\n"+
			"invalid longitude: must be at least -180 (min)", err.Error())
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
		var failed interface{ Has(field, rule string) bool }
		require.ErrorAs(t, err, &failed)
		assert.True(t, failed.Has("coordinates", "pair"))
	})
}

func TestCoordinates_ZeroValue(t *testing.T) {
	t.Parallel()

	// Arrange
	var zero geo.Coordinates
	constructed, err := geo.NewCoordinates("40.416775", "-3.70379")
	require.NoError(t, err)

	// Act
	_, textErr := zero.MarshalText()
	decoded := constructed
	nullErr := json.Unmarshal([]byte("null"), &decoded)

	// Assert
	require.ErrorIs(t, textErr, errors.ErrUnsupported)
	require.NoError(t, nullErr)
	assert.False(t, decoded.IsZero(), "null is a no-op for a text unmarshaler, so the value stays")
}
