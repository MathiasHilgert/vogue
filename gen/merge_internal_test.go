package gen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeConsts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "leaves a single constant alone", in: []string{"const maxParam = 120"}, want: []string{"const maxParam = 120"}},
		{
			name: "folds the constants of every rule into one block",
			in:   []string{"const minParam = 1", "const maxParamCoef = 905\nconst maxParamScale = 1"},
			want: []string{"const (\n\tminParam = 1\n\tmaxParamCoef = 905\n\tmaxParamScale = 1\n)"},
		},
		{
			name: "keeps declarations of another shape as they are",
			in:   []string{"const minParam = 1", "var cache = map[string]int{}"},
			want: []string{"const minParam = 1", "var cache = map[string]int{}"},
		},
		{name: "has nothing to fold", in: nil, want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got := mergeConsts(tc.in)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}
