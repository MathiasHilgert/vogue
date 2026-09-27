package validation_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/MathiasHilgert/vogue/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrInvalid(t *testing.T) {
	t.Parallel()

	t.Run("a field error is an invalid value", func(t *testing.T) {
		t.Parallel()

		// Arrange
		err := validation.FieldError{Field: "title", Rule: "required", Message: "title is required"}

		// Act & Assert
		require.ErrorIs(t, err, validation.ErrInvalid)
	})

	t.Run("a notification with failures is an invalid value", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var notification validation.Notification
		notification.Add(validation.FieldError{Field: "title", Rule: "required", Message: "title is required"})

		// Act
		err := fmt.Errorf("creating the tab: %w", notification.ErrOrNil())

		// Assert
		require.ErrorIs(t, err, validation.ErrInvalid)
	})

	t.Run("an unrelated error is not an invalid value", func(t *testing.T) {
		t.Parallel()

		// Arrange
		err := errors.New("connection refused")

		// Act & Assert
		assert.NotErrorIs(t, err, validation.ErrInvalid)
	})

	t.Run("the sentinel does not match a field error by example", func(t *testing.T) {
		t.Parallel()

		// Arrange
		err := validation.FieldError{Field: "title", Rule: "required", Message: "title is required"}

		// Act & Assert
		assert.NotErrorIs(t, validation.ErrInvalid, err)
	})
}

func TestSourceSentinels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		sentinel error
	}{
		{name: "an unsupported source", sentinel: validation.ErrUnsupportedSource},
		{name: "a lossy source", sentinel: validation.ErrLossySource},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			err := fmt.Errorf("vogue: cannot scan float64 into Rate: %w", testCase.sentinel)

			// Act & Assert
			require.ErrorIs(t, err, testCase.sentinel)
			require.NotErrorIs(t, err, validation.ErrInvalid, "a driver problem is not a validation failure")
			assert.NotEmpty(t, testCase.sentinel.Error())
		})
	}
}

func TestNotification_HasErrors(t *testing.T) {
	t.Parallel()

	// Arrange
	var notification validation.Notification

	// Act
	before := notification.HasErrors()
	notification.Add(validation.FieldError{Field: "title", Rule: "required", Message: "title is required"})

	// Assert
	assert.False(t, before)
	assert.True(t, notification.HasErrors())
}

func TestNotification_Reject(t *testing.T) {
	t.Parallel()

	// Arrange
	var notification validation.Notification

	// Act
	notification.Reject("title", "min", "1", "", "title must be at least 1")

	// Assert
	require.Equal(t, 1, notification.Len())
	assert.Equal(t, validation.FieldError{
		Field: "title", Rule: "min", Param: "1", Value: "", Message: "title must be at least 1",
	}, notification.Errors()[0])
}

func TestNotification_Reject_zeroAllocationsOnceGrown(t *testing.T) {
	// Arrange
	var notification validation.Notification
	notification.Reject("title", "required", "", "", "title is required")

	// Act
	allocs := testing.AllocsPerRun(100, func() {
		notification.Reset()
		notification.Reject("title", "required", "", "", "title is required")
	})

	// Assert
	assert.Zero(t, allocs)
}

func TestNotification_Collect(t *testing.T) {
	t.Parallel()

	t.Run("merges every failure of a validation error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var inner validation.Notification
		inner.Reject("latitude", "max", "90", "91", "latitude must be at most 90")
		inner.Reject("latitude", "scale", "6", "91", "latitude must have at most 6 decimal places")

		var notification validation.Notification

		// Act
		err := notification.Collect(fmt.Errorf("building coordinates: %w", inner.ErrOrNil()))

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 2, notification.Len())
	})

	t.Run("adds a lone field error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var notification validation.Notification

		// Act
		err := notification.Collect(validation.FieldError{Field: "longitude", Rule: "min", Message: "longitude must be at least -180"})

		// Assert
		require.NoError(t, err)
		assert.Equal(t, 1, notification.Len())
	})

	t.Run("ignores nil", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var notification validation.Notification

		// Act
		err := notification.Collect(nil)

		// Assert
		require.NoError(t, err)
		assert.False(t, notification.HasErrors())
	})

	t.Run("keeps the other errors of a join", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var inner validation.Notification
		inner.Reject("latitude", "max", "90", "91", "latitude must be at most 90")
		driverFailure := errors.New("connection reset")

		var notification validation.Notification

		// Act
		err := notification.Collect(errors.Join(inner.ErrOrNil(), driverFailure))

		// Assert
		require.ErrorIs(t, err, driverFailure, "a join's non-validation error must not be swallowed")
		require.NotErrorIs(t, err, validation.ErrInvalid, "the validation failure is merged, not returned")
		assert.Equal(t, 1, notification.Len())
	})

	t.Run("treats an empty notification as no error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var notification validation.Notification

		// Act
		err := notification.Collect(&validation.Notification{})

		// Assert
		require.NoError(t, err)
		assert.False(t, notification.HasErrors())
	})

	t.Run("hands back an error that is not a validation failure", func(t *testing.T) {
		t.Parallel()

		// Arrange
		var notification validation.Notification
		boom := errors.New("minting failed")

		// Act
		err := notification.Collect(boom)

		// Assert
		require.ErrorIs(t, err, boom)
		assert.False(t, notification.HasErrors())
	})
}
