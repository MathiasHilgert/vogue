package parse_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIDStrategyZeroValueIsUUIDv7(t *testing.T) {
	t.Parallel()

	// Arrange / Act / Assert.
	var zero parse.IDStrategy
	assert.Equal(t, parse.IDUUIDv7, zero)
	assert.Equal(t, "uuid7", zero.String())
}

func TestIDStrategyString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   parse.IDStrategy
		want string
	}{
		{name: "uuid7", in: parse.IDUUIDv7, want: "uuid7"},
		{name: "uuid4", in: parse.IDUUIDv4, want: "uuid4"},
		{name: "int64", in: parse.IDInt64, want: "int64"},
		{name: "unknown", in: parse.IDStrategy(9), want: "IDStrategy(9)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act / Assert.
			assert.Equal(t, tt.want, tt.in.String())
		})
	}
}

func TestIDDirectiveStrategy(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want parse.IDStrategy
	}{
		{name: "absent defaults to uuid7", src: "package tab\n//vogue:id TabID\n", want: parse.IDUUIDv7},
		{name: "explicit uuid7", src: "package tab\n//vogue:id TabID uuid7\n", want: parse.IDUUIDv7},
		{name: "explicit uuid4", src: "package tab\n//vogue:id TabID uuid4\n", want: parse.IDUUIDv4},
		{name: "explicit int64", src: "package tab\n//vogue:id TabID int64\n", want: parse.IDInt64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act.
			pkg, err := parseSrc(t, tt.src)

			// Assert.
			require.NoError(t, err)
			directives := pkg.Directives()
			require.Len(t, directives, 1)
			assert.Equal(t, tt.want, directives[0].Strategy)
			assert.Empty(t, directives[0].Rules)
		})
	}
}

func TestIDDirectiveStrategyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		src    string
		needle string
		want   string
	}{
		{
			name:   "unknown strategy suggests the nearest",
			src:    "package tab\n//vogue:id TabID uuid8\n",
			needle: "uuid8",
			want:   `unknown id strategy "uuid8" (did you mean "uuid4"?)`,
		},
		{
			name:   "unknown strategy without a near match lists them",
			src:    "package tab\n//vogue:id TabID min=1\n",
			needle: "min=1",
			want:   `unknown id strategy "min=1" (want one of: uuid7, uuid4, int64)`,
		},
		{
			name:   "extra token",
			src:    "package tab\n//vogue:id TabID uuid4 int64\n",
			needle: "int64",
			want:   `kind id takes at most one strategy, got "int64"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act.
			pkg, err := parseSrc(t, tt.src)

			// Assert.
			require.Error(t, err)
			assert.Nil(t, pkg)
			assert.Equal(t, at(t, tt.src, tt.needle)+": "+tt.want, err.Error())
		})
	}
}
