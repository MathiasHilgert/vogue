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
		var n validation.Notification
		n.Add(validation.FieldError{Field: "title", Rule: "required", Message: "title is required"})

		// Act
		err := fmt.Errorf("creating the tab: %w", n.ErrOrNil())

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

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Arrange
			err := fmt.Errorf("vogue: cannot scan float64 into Rate: %w", tc.sentinel)

			// Act & Assert
			require.ErrorIs(t, err, tc.sentinel)
			require.NotErrorIs(t, err, validation.ErrInvalid, "a driver problem is not a validation failure")
			assert.NotEmpty(t, tc.sentinel.Error())
		})
	}
}

func TestNotification_HasErrors(t *testing.T) {
	t.Parallel()

	// Arrange
	var n validation.Notification

	// Act
	before := n.HasErrors()
	n.Add(validation.FieldError{Field: "title", Rule: "required", Message: "title is required"})

	// Assert
	assert.False(t, before)
	assert.True(t, n.HasErrors())
}
