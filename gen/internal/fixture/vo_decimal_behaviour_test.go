package fixture_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/MathiasHilgert/vogue/gen/internal/fixture"
	"github.com/govalues/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustDecimal parses a decimal literal a test is written against.
func mustDecimal(t *testing.T, s string) decimal.Decimal {
	t.Helper()

	d, err := decimal.Parse(s)
	require.NoError(t, err)
	return d
}

func TestWeight(t *testing.T) {
	t.Parallel()

	t.Run("accepts a weight inside the bound and the scale", func(t *testing.T) {
		t.Parallel()

		// Act
		got, err := fixture.NewWeight(mustDecimal(t, "12.750"))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "12.750", got.String())
		assert.Equal(t, 3, got.Decimal().Scale())
	})

	t.Run("rejects a weight below the bound", func(t *testing.T) {
		t.Parallel()

		// Act
		got, err := fixture.NewWeight(mustDecimal(t, "-0.25"))

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "weight", "min"), "%v", err)
		assert.True(t, got.IsZero())
	})

	t.Run("rejects a weight measured finer than the scale reads", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := fixture.NewWeight(mustDecimal(t, "0.1234"))

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "weight", "scale"), "%v", err)
	})

	t.Run("an unreadable representation fails the decimal rule", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := fixture.NewWeightFromString("one and a half")

		// Assert
		require.Error(t, err)
		assert.True(t, rejectedBy(err, "weight", "decimal"), "%v", err)
	})

	t.Run("equality compares the number and not the scale it was written at", func(t *testing.T) {
		t.Parallel()

		// Arrange
		plain, err := fixture.NewWeightFromString("1.5")
		require.NoError(t, err)
		padded, err := fixture.NewWeightFromString("1.500")
		require.NoError(t, err)

		// Act & Assert
		assert.True(t, plain.Equal(padded), "1.5 and 1.500 are the same weight")
		assert.NotEqual(t, plain.String(), padded.String(), "and they are still stored as written")
	})

	t.Run("crosses a JSON boundary as a string", func(t *testing.T) {
		t.Parallel()

		// Arrange
		want, err := fixture.NewWeightFromString("0.125")
		require.NoError(t, err)

		// Act
		body, err := json.Marshal(want)
		require.NoError(t, err)

		var got fixture.Weight
		err = json.Unmarshal(body, &got)

		// Assert
		require.NoError(t, err)
		assert.JSONEq(t, `"0.125"`, string(body))
		assert.True(t, want.Equal(got))
	})

	t.Run("reads a numeric column delivered as text or as a whole number", func(t *testing.T) {
		t.Parallel()

		// Arrange
		cases := map[string]any{
			"text":  "2.500",
			"bytes": []byte("2.500"),
			"int64": int64(2),
		}

		for name, src := range cases {
			t.Run(name, func(t *testing.T) {
				t.Parallel()

				// Act
				var got fixture.Weight
				err := got.Scan(src)

				// Assert
				require.NoError(t, err)
				assert.False(t, got.IsZero())
			})
		}
	})

	t.Run("refuses a binary float source", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var got fixture.Weight

		// Act
		err := got.Scan(0.1)

		// Assert
		require.Error(t, err)
		require.ErrorIs(t, err, errors.ErrUnsupported)
		assert.Contains(t, err.Error(), "binary float")
		assert.True(t, got.IsZero())
	})

	t.Run("a stored weight survives the SQL round trip unrounded", func(t *testing.T) {
		t.Parallel()

		// Arrange
		want, err := fixture.NewWeightFromString("0.125")
		require.NoError(t, err)

		// Act
		stored, err := want.Value()
		require.NoError(t, err)

		var got fixture.Weight
		err = got.Scan(stored)

		// Assert
		require.NoError(t, err)
		assert.Equal(t, "0.125", stored)
		assert.Equal(t, want.String(), got.String())
	})
}
