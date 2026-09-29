package validation_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/examples/validation"
)

func TestValidation_Err(t *testing.T) {
	t.Parallel()

	t.Run("is nil until something is added", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var failures validation.Validation

		// Act
		err := failures.Err()

		// Assert
		require.NoError(t, err)
	})

	t.Run("lists every failure, never the value", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var failures validation.Validation
		failures.Add("slug", "required", "is required")
		failures.Add("title", "max", "length must be at most 120")

		// Act
		err := failures.Err()

		// Assert
		require.Error(t, err)
		assert.Equal(t, "invalid slug: is required (required); invalid title: length must be at most 120 (max)", err.Error())
	})
}

// TestValidation_HappyPathAllocatesNothing is not parallel: AllocsPerRun cannot
// run beside parallel tests.
func TestValidation_HappyPathAllocatesNothing(t *testing.T) { //nolint:paralleltest // see above.
	// Act
	allocations := testing.AllocsPerRun(100, func() {
		var failures validation.Validation

		err := failures.Err()
		if err != nil {
			t.Fatal(err)
		}
	})

	// Assert
	assert.Zero(t, allocations)
}

func TestError_Has(t *testing.T) {
	t.Parallel()

	// Arrange
	var failures validation.Validation
	failures.Add("slug", "required", "is required")

	var failed interface{ Has(field, rule string) bool }
	require.ErrorAs(t, failures.Err(), &failed)

	// Act and Assert
	assert.True(t, failed.Has("slug", "required"))
	assert.False(t, failed.Has("slug", "max"), "another rule of the same field")
	assert.False(t, failed.Has("title", "required"), "the same rule of another field")
}
