package vogue_test

import (
	"errors"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFieldError_Error(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		err  vogue.FieldError
		want string
	}{
		{
			name: "with param",
			err:  vogue.FieldError{Field: "title", Rule: "min", Param: "1", Message: "must be between 1 and 120 characters"},
			want: `title: must be between 1 and 120 characters (rule "min", param "1")`,
		},
		{
			name: "without param",
			err:  vogue.FieldError{Field: "email", Rule: "email", Message: "must be a valid email address"},
			want: `email: must be a valid email address (rule "email")`,
		},
		{
			name: "value is not rendered",
			err:  vogue.FieldError{Field: "email", Rule: "email", Value: "nope", Message: "must be a valid email address"},
			want: `email: must be a valid email address (rule "email")`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := tc.err.Error()

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFieldError_Code(t *testing.T) {
	// Arrange
	err := vogue.FieldError{Field: "title", Rule: "min"}

	// Act
	got := err.Code()

	// Assert
	assert.Equal(t, "title.min", got)
}

func TestFieldError_Is(t *testing.T) {
	// Arrange
	err := vogue.FieldError{Field: "title", Rule: "min", Param: "1", Message: "too short"}

	cases := []struct {
		name   string
		target error
		want   bool
	}{
		{name: "identical", target: vogue.FieldError{Field: "title", Rule: "min", Param: "1", Message: "too short"}, want: true},
		{name: "same field and rule", target: vogue.FieldError{Field: "title", Rule: "min"}, want: true},
		{name: "rule only", target: vogue.FieldError{Rule: "min"}, want: true},
		{name: "field only", target: vogue.FieldError{Field: "title"}, want: true},
		{name: "other rule", target: vogue.FieldError{Rule: "max"}, want: false},
		{name: "other field", target: vogue.FieldError{Field: "email"}, want: false},
		{name: "other param", target: vogue.FieldError{Rule: "min", Param: "2"}, want: false},
		{name: "foreign error", target: errors.New("boom"), want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := errors.Is(err, tc.target)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestFieldError_As(t *testing.T) {
	// Arrange
	want := vogue.FieldError{Field: "title", Rule: "min", Message: "too short"}
	var err error = want

	// Act
	var got vogue.FieldError
	ok := errors.As(err, &got)

	// Assert
	require.True(t, ok)
	assert.Equal(t, want, got)
}
