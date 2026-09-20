package parse_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		in         string
		wantKind   vogue.Kind
		wantName   string
		wantTokens []string
	}{
		{
			name:       "string with rules",
			in:         "//vogue:string  Title      required trim min=1 max=120",
			wantKind:   vogue.String,
			wantName:   "Title",
			wantTokens: []string{"required", "trim", "min=1", "max=120"},
		},
		{
			name:       "quoted param keeps spaces and equals",
			in:         `//vogue:string Code regex="^[a-z]+=[0-9] +$" upper`,
			wantKind:   vogue.String,
			wantName:   "Code",
			wantTokens: []string{"regex=^[a-z]+=[0-9] +$", "upper"},
		},
		{
			name:       "quoted param honours go escapes",
			in:         `//vogue:string Code prefix="a\tb\"c"`,
			wantKind:   vogue.String,
			wantName:   "Code",
			wantTokens: []string{"prefix=a\tb\"c"},
		},
		{
			name:       "int without rules",
			in:         "//vogue:int Covers",
			wantKind:   vogue.Int,
			wantName:   "Covers",
			wantTokens: nil,
		},
		{
			name:       "enum list is one token",
			in:         "//vogue:enum TabStatus open,closed,voided",
			wantKind:   vogue.Enum,
			wantName:   "TabStatus",
			wantTokens: []string{"open,closed,voided"},
		},
		{
			name:       "id",
			in:         "//vogue:id TabID",
			wantKind:   vogue.ID,
			wantName:   "TabID",
			wantTokens: nil,
		},
		{
			name:       "trailing whitespace is ignored",
			in:         "//vogue:string Title   ",
			wantKind:   vogue.String,
			wantName:   "Title",
			wantTokens: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act.
			kind, name, tokens, err := parse.Line(tt.in)

			// Assert.
			require.NoError(t, err)
			assert.Equal(t, tt.wantKind, kind)
			assert.Equal(t, tt.wantName, name)
			assert.Equal(t, tt.wantTokens, tokens)
		})
	}
}

func TestLineErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		wantErr string
	}{
		{name: "not a directive", in: "// ordinary comment", wantErr: `not a vogue directive`},
		{name: "missing kind", in: "//vogue:", wantErr: `missing kind after "//vogue:"`},
		{name: "unknown kind", in: "//vogue:strng Title", wantErr: `unknown kind "strng" (did you mean "string"?)`},
		{name: "missing name", in: "//vogue:string", wantErr: `missing type name after kind "string"`},
		{name: "unterminated quote", in: `//vogue:string Title regex="^a`, wantErr: `unterminated quoted parameter`},
		{name: "bad escape", in: `//vogue:string Title prefix="a\q"`, wantErr: `invalid quoted parameter "\"a\\q\""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act.
			_, _, _, err := parse.Line(tt.in)

			// Assert.
			require.Error(t, err)
			assert.Equal(t, tt.wantErr, err.Error())
		})
	}
}
