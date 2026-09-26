package vogue_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/validation"
	"github.com/stretchr/testify/require"
)

// TestAliases pins the backwards-compatible names: code written against
// vogue.FieldError and vogue.Notification before the runtime moved to package
// validation keeps compiling and keeps matching with errors.Is.
func TestAliases(t *testing.T) {
	t.Parallel()

	// Arrange
	var notification vogue.Notification
	notification.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "title is required"})

	// Act
	err := notification.ErrOrNil()

	// Assert
	require.ErrorIs(t, err, validation.FieldError{Rule: "required"})
	require.ErrorIs(t, err, vogue.ErrInvalid)
}
