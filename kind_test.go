package vogue_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKind_String(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		kind vogue.Kind
		want string
	}{
		{name: "string", kind: vogue.String, want: "string"},
		{name: "int", kind: vogue.Int, want: "int"},
		{name: "enum", kind: vogue.Enum, want: "enum"},
		{name: "id", kind: vogue.ID, want: "id"},
		{name: "unknown", kind: vogue.Kind(200), want: "Kind(200)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := tc.kind.String()

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseKind(t *testing.T) {
	// Arrange
	cases := []struct {
		name    string
		in      string
		want    vogue.Kind
		wantErr bool
	}{
		{name: "string", in: "string", want: vogue.String},
		{name: "int", in: "int", want: vogue.Int},
		{name: "enum", in: "enum", want: vogue.Enum},
		{name: "id", in: "id", want: vogue.ID},
		{name: "decimal", in: "decimal", want: vogue.Decimal},
		{name: "unknown", in: "money", wantErr: true},
		{name: "empty", in: "", wantErr: true},
		{name: "case sensitive", in: "String", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got, err := vogue.ParseKind(tc.in)

			// Assert
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "vogue: unknown kind")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestKinds(t *testing.T) {
	// Arrange
	set := vogue.Kinds(vogue.String, vogue.Int)

	// Act
	// (construction is the action)

	// Assert
	assert.True(t, set.Has(vogue.String))
	assert.True(t, set.Has(vogue.Int))
	assert.False(t, set.Has(vogue.Enum))
	assert.False(t, set.Has(vogue.ID))
}

func TestKindSet_Has(t *testing.T) {
	t.Run("zero value has nothing", func(t *testing.T) {
		// Arrange
		var set vogue.KindSet

		// Act
		got := set.Has(vogue.String)

		// Assert
		assert.False(t, got)
		assert.True(t, set.Empty())
	})

	t.Run("duplicates are idempotent", func(t *testing.T) {
		// Arrange
		set := vogue.Kinds(vogue.Enum, vogue.Enum)

		// Act
		got := set.Kinds()

		// Assert
		assert.Equal(t, []vogue.Kind{vogue.Enum}, got)
	})
}

func TestKindSet_String(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		set  vogue.KindSet
		want string
	}{
		{name: "empty", set: vogue.KindSet(0), want: ""},
		{name: "single", set: vogue.Kinds(vogue.Int), want: "int"},
		{name: "declaration order is normalized", set: vogue.Kinds(vogue.ID, vogue.String), want: "string, id"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := tc.set.String()

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}
