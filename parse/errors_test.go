package parse_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorRendering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *parse.Error
		want string
	}{
		{
			name: "position and hint",
			err: &parse.Error{
				Pos:  token.Position{Filename: "vo.go", Line: 12, Column: 1},
				Msg:  `unknown rule "mni" for kind string`,
				Hint: `did you mean "min"?`,
			},
			want: `vo.go:12:1: unknown rule "mni" for kind string (did you mean "min"?)`,
		},
		{
			name: "position without hint",
			err:  &parse.Error{Pos: token.Position{Filename: "vo.go", Line: 3, Column: 5}, Msg: "boom"},
			want: "vo.go:3:5: boom",
		},
		{
			name: "no position",
			err:  &parse.Error{Msg: "not a vogue directive"},
			want: "not a vogue directive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act / Assert.
			assert.Equal(t, tt.want, tt.err.Error())
		})
	}
}

func TestErrorsAreSortedByPosition(t *testing.T) {
	t.Parallel()

	// Arrange.
	fset := token.NewFileSet()
	files := make([]*ast.File, 0, 2)
	for _, name := range []string{"z.go", "a.go"} {
		file, err := parser.ParseFile(fset, name, "package tab\n//vogue:string Title zzzzzz\n", parser.ParseComments)
		require.NoError(t, err)
		files = append(files, file)
	}

	// Act.
	_, err := parse.Files(fset, files, testRules(t))

	// Assert.
	require.Error(t, err)
	var errs parse.Errors
	require.ErrorAs(t, err, &errs)
	require.Len(t, errs, 2)
	assert.Equal(t, "a.go", errs[0].Pos.Filename)
	assert.Equal(t, "z.go", errs[1].Pos.Filename)
}

func TestFilesRejectsMixedPackages(t *testing.T) {
	t.Parallel()

	// Arrange.
	fset := token.NewFileSet()
	a, err := parser.ParseFile(fset, "a.go", "package tab\n", parser.ParseComments)
	require.NoError(t, err)
	b, err := parser.ParseFile(fset, "b.go", "package other\n", parser.ParseComments)
	require.NoError(t, err)

	// Act.
	_, err = parse.Files(fset, []*ast.File{a, b}, testRules(t))

	// Assert.
	require.Error(t, err)
	assert.Equal(t, `b.go:1:9: found packages "tab" and "other" in the same directory`, err.Error())
}

func TestFilesAcceptsNilRuleSet(t *testing.T) {
	t.Parallel()

	// Arrange.
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "vo.go", "package tab\n//vogue:string Title min=1\n", parser.ParseComments)
	require.NoError(t, err)

	// Act.
	_, err = parse.Files(fset, []*ast.File{file}, nil)

	// Assert.
	require.Error(t, err)
	assert.Equal(t, `vo.go:2:22: unknown rule "min" for kind string`, err.Error())
}
