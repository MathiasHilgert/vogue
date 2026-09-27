package gen_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shortAllowed are the identifiers under three characters generated code may
// declare: the conventional names no reader mistakes for an abbreviation.
var shortAllowed = map[string]bool{"ok": true, "id": true, "ctx": true, "err": true, "_": true}

// TestGenerate_NoAbbreviatedIdentifiers holds generated code to the
// no-abbreviation rule of the projects that consume it: every identifier it
// declares — types, functions, receivers, parameters, results, variables,
// constants, fields and import names — is at least three characters long,
// except ok, id, ctx and err. A generated test may also name its *testing.T
// t, which is how every Go test is written.
func TestGenerate_NoAbbreviatedIdentifiers(t *testing.T) {
	goldens, err := filepath.Glob(filepath.Join("testdata", "strict", "*", "*_vogue*.go"))
	require.NoError(t, err)
	goldens = append(goldens,
		filepath.Join("testdata", "tab", "want", "vo_vogue.go"),
		filepath.Join("testdata", "tab", "want", "vo_vogue_test.go"),
		filepath.Join("internal", "fixture", "vo_vogue.go"),
		filepath.Join("internal", "fixture", "vo_vogue_test.go"),
		filepath.Join("..", "rules", "internal", "catalogue", "vo_vogue.go"),
		filepath.Join("..", "rules", "internal", "catalogue", "vo_vogue_test.go"),
		filepath.Join("..", "examples", "composite", "geo", "vo_vogue.go"),
		filepath.Join("..", "examples", "composite", "geo", "coordinates.go"),
	)

	for _, golden := range goldens {
		t.Run(golden, func(t *testing.T) {
			// Arrange
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, golden, nil, 0)
			require.NoError(t, err)
			allowed := func(name string) bool {
				return shortAllowed[name] || (name == "t" && strings.HasSuffix(golden, "_test.go"))
			}

			// Act
			var short []string
			for _, ident := range declaredIdentifiers(file) {
				if len(ident.Name) < 3 && !allowed(ident.Name) {
					short = append(short, fset.Position(ident.Pos()).String()+" "+ident.Name)
				}
			}

			// Assert
			assert.Empty(t, short, "generated code declares abbreviated identifiers")
		})
	}
}

// declaredIdentifiers returns every identifier a file declares.
func declaredIdentifiers(file *ast.File) []*ast.Ident {
	var out []*ast.Ident
	fields := func(list *ast.FieldList) {
		if list == nil {
			return
		}
		for _, field := range list.List {
			out = append(out, field.Names...)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			fields(node.Recv)
			out = append(out, node.Name)
		case *ast.FuncType:
			fields(node.Params)
			fields(node.Results)
		case *ast.StructType:
			fields(node.Fields)
		case *ast.TypeSpec:
			out = append(out, node.Name)
		case *ast.ValueSpec:
			out = append(out, node.Names...)
		case *ast.ImportSpec:
			if node.Name != nil {
				out = append(out, node.Name)
			}
		case *ast.AssignStmt:
			if node.Tok == token.DEFINE {
				for _, lhs := range node.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						out = append(out, ident)
					}
				}
			}
		case *ast.RangeStmt:
			for _, expr := range []ast.Expr{node.Key, node.Value} {
				if ident, ok := expr.(*ast.Ident); ok {
					out = append(out, ident)
				}
			}
		}
		return true
	})
	return out
}
