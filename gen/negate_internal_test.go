package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNegate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "inverts an equality", in: `v != ""`, want: `v == ""`},
		{name: "inverts an ordering", in: `utf8.RuneCountInString(v) >= minParam`, want: `utf8.RuneCountInString(v) < minParam`},
		{name: "negates a call without parentheses", in: `fn.Email(v)`, want: `!fn.Email(v)`},
		{name: "drops a double negation", in: `!strings.Contains(v, "x")`, want: `strings.Contains(v, "x")`},
		{name: "applies De Morgan to a disjunction", in: `v == "a" || v == "b"`, want: `v != "a" && v != "b"`},
		{name: "applies De Morgan to a conjunction", in: `v > 0 && v < 9`, want: `v <= 0 || v >= 9`},
		{
			name: "needs no parentheses where the new conjunction binds tighter",
			in:   `(v == "a" || v == "b") && v != "c"`,
			want: `v != "a" && v != "b" || v == "c"`,
		},
		{
			name: "parenthesises a new disjunction inside a conjunction",
			in:   `v > 0 && v < 9 || v == 42`,
			want: `(v <= 0 || v >= 9) && v != 42`,
		},
		{name: "unwraps parentheses", in: `(v > 0)`, want: `v <= 0`},
		{name: "compares a method result", in: `v.Cmp(maxBound) <= 0`, want: `v.Cmp(maxBound) > 0`},
		{name: "negates an identifier", in: `ok`, want: `!ok`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got, err := negate(tc.in)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("reports an expression that does not parse", func(t *testing.T) {
		t.Parallel()

		// Act
		_, err := negate(`v >=`)

		// Assert
		require.Error(t, err)
	})
}
