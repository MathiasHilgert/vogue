package vogue_test

import (
	"errors"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotification_Add(t *testing.T) {
	t.Run("zero value is usable", func(t *testing.T) {
		// Arrange
		var n vogue.Notification

		// Act
		n.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "is required"})

		// Assert
		assert.Equal(t, 1, n.Len())
		assert.True(t, n.HasErrors())
	})

	t.Run("empty notification reports no errors", func(t *testing.T) {
		// Arrange
		var n vogue.Notification

		// Act
		got := n.HasErrors()

		// Assert
		assert.False(t, got)
		assert.Equal(t, 0, n.Len())
		assert.Empty(t, n.Errors())
	})
}

func TestNotification_Addf(t *testing.T) {
	// Arrange
	var n vogue.Notification

	// Act
	n.Addf("title", "min", "3", "ab", "must be at least %d characters", 3)

	// Assert
	require.Equal(t, 1, n.Len())
	assert.Equal(t, vogue.FieldError{
		Field:   "title",
		Rule:    "min",
		Param:   "3",
		Value:   "ab",
		Message: "must be at least 3 characters",
	}, n.Errors()[0])
}

func TestNotification_Merge(t *testing.T) {
	// Arrange
	var dst, src vogue.Notification
	dst.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "is required"})
	src.Add(vogue.FieldError{Field: "email", Rule: "email", Message: "must be a valid email address"})

	// Act
	dst.Merge(&src)

	// Assert
	require.Equal(t, 2, dst.Len())
	assert.Equal(t, "title", dst.Errors()[0].Field)
	assert.Equal(t, "email", dst.Errors()[1].Field)
	assert.Equal(t, 1, src.Len(), "source must not be mutated")
}

func TestNotification_Merge_nilIsNoop(t *testing.T) {
	// Arrange
	var dst vogue.Notification
	dst.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "is required"})

	// Act
	dst.Merge(nil)

	// Assert
	assert.Equal(t, 1, dst.Len())
}

func TestNotification_Errors(t *testing.T) {
	// Arrange
	var n vogue.Notification
	n.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "is required"})

	// Act
	got := n.Errors()
	got[0].Field = "mutated"

	// Assert
	assert.Equal(t, "title", n.Errors()[0].Field, "Errors must return a defensive copy")
}

func TestNotification_Field(t *testing.T) {
	// Arrange
	var n vogue.Notification
	n.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "is required"})
	n.Add(vogue.FieldError{Field: "email", Rule: "email", Message: "must be a valid email address"})
	n.Add(vogue.FieldError{Field: "title", Rule: "min", Param: "1", Message: "too short"})

	// Act
	got := n.Field("title")

	// Assert
	require.Len(t, got, 2)
	assert.Equal(t, "required", got[0].Rule)
	assert.Equal(t, "min", got[1].Rule)
	assert.Empty(t, n.Field("missing"))
}

func TestNotification_Error(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		errs []vogue.FieldError
		want string
	}{
		{
			name: "empty",
			want: "no validation errors",
		},
		{
			name: "single",
			errs: []vogue.FieldError{{Field: "title", Rule: "required", Message: "is required"}},
			want: "1 validation error:\n  - title: is required (rule \"required\")",
		},
		{
			name: "multiple",
			errs: []vogue.FieldError{
				{Field: "title", Rule: "min", Param: "1", Message: "too short"},
				{Field: "email", Rule: "email", Message: "must be a valid email address"},
			},
			want: "2 validation errors:\n  - title: too short (rule \"min\", param \"1\")\n  - email: must be a valid email address (rule \"email\")",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			var n vogue.Notification
			for _, e := range tc.errs {
				n.Add(e)
			}

			// Act
			got := n.Error()

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestNotification_ErrOrNil(t *testing.T) {
	t.Run("empty returns a truly nil error", func(t *testing.T) {
		// Arrange
		var n vogue.Notification

		// Act
		err := n.ErrOrNil()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, error(nil), err, "must not be a typed nil trapped in an interface")
	})

	t.Run("non empty returns the notification", func(t *testing.T) {
		// Arrange
		var n vogue.Notification
		n.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "is required"})

		// Act
		err := n.ErrOrNil()

		// Assert
		require.Error(t, err)
		assert.Same(t, &n, err)
	})
}

func TestNotification_Unwrap(t *testing.T) {
	// Arrange
	var n vogue.Notification
	n.Add(vogue.FieldError{Field: "title", Rule: "min", Param: "1", Message: "too short"})
	n.Add(vogue.FieldError{Field: "email", Rule: "email", Message: "must be a valid email address"})
	err := n.ErrOrNil()

	t.Run("errors.Is finds a rule", func(t *testing.T) {
		// Act & Assert
		require.ErrorIs(t, err, vogue.FieldError{Rule: "email"})
		assert.NotErrorIs(t, err, vogue.FieldError{Rule: "max"})
	})

	t.Run("errors.As extracts the first field error", func(t *testing.T) {
		// Act
		var got vogue.FieldError
		ok := errors.As(err, &got)

		// Assert
		require.True(t, ok)
		assert.Equal(t, "title.min", got.Code())
	})
}

func TestNotification_Add_zeroAllocationsWhenEmpty(t *testing.T) {
	// Arrange
	var n vogue.Notification

	// Act
	allocs := testing.AllocsPerRun(100, func() {
		n.Reset()
		n.Add(vogue.FieldError{Field: "title", Rule: "required", Message: "is required"})
	})

	// Assert
	assert.Zero(t, allocs, "Reset must keep capacity so repeated use does not allocate")
}
