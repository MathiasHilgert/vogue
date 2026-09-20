package gen_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/require"
)

// parseSource parses directives from an in-memory source body, so a test can
// state the exact directive it is about without owning a testdata directory.
func parseSource(t *testing.T, body string, rules *vogue.RuleSet) *parse.Package {
	t.Helper()

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "vo.go", "package tab\n\n"+body, parser.ParseComments)
	require.NoError(t, err)

	pkg, err := parse.Files(fset, []*ast.File{file}, rules)
	require.NoError(t, err)
	return pkg
}

// filesOf runs the whole generator and returns the first error it reports,
// whichever stage produced it.
func filesOf(t *testing.T, opts gen.Options) ([]gen.OutFile, error) {
	t.Helper()

	g, err := gen.New(opts)
	if err != nil {
		return nil, err
	}
	return g.Files()
}
