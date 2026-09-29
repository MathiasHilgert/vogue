package gen_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/MathiasHilgert/vogue/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// strictVariants are the golden outputs generated from testdata/strict/vo.go.
// CI lints both with gen/testdata/strict/golangci.yml, a copy of a real
// consumer's configuration with nothing excluded for generated code, and runs
// the generated tests of both.
var strictVariants = []struct {
	dir  string
	opts gen.Options
}{
	// domain is what a hexagonal domain package holds: no database/sql/driver,
	// which the consumer's depguard forbids there, and the JSONSchema method
	// the HTTP adapter publishes.
	{dir: "domain", opts: gen.Options{OmitSQL: true, Schema: true}},
	// persistence is the same directives with the SQL codec.
	{dir: "persistence", opts: gen.Options{}},
}

func TestGenerate_Strict(t *testing.T) {
	for _, variant := range strictVariants {
		t.Run(variant.dir+" matches the committed golden output", func(t *testing.T) {
			// Arrange
			pkg, err := parse.Dir(filepath.Join("testdata", "strict"), rules.MustSet())
			require.NoError(t, err)
			opts := variant.opts
			opts.Package, opts.Rules = pkg, rules.MustSet()

			// Act
			files, err := filesOf(t, opts)

			// Assert
			require.NoError(t, err)
			require.Len(t, files, 4, "two source files, each with its code and its test")
			for _, file := range files {
				golden := filepath.Join("testdata", "strict", variant.dir, filepath.Base(file.Path))
				if *update {
					require.NoError(t, os.MkdirAll(filepath.Dir(golden), 0o750))
					require.NoError(t, os.WriteFile(golden, file.Content, 0o600))
				}
				want, err := os.ReadFile(golden)
				require.NoError(t, err)
				assert.Equal(t, string(want), string(file.Content), "%s has drifted; re-run with -update", golden)
			}
		})
	}
}

// TestGenerate_MethodsOnly pins the shape of the generated API: the only
// package-level functions are constructors, and the only package-level
// variables are compiled patterns. Everything else — enum members, parsing, codecs — is a
// method, so a generated package adds no free-standing identifiers beyond the
// types it declares and their New constructors. The one package-level
// variable allowed is a compiled pattern: a regex rule compiles its expression
// once, at package initialisation, into an unexported `<type>Pattern` variable.
func TestGenerate_MethodsOnly(t *testing.T) {
	goldens := []string{
		filepath.Join("testdata", "strict", "domain", "vo_vogue.go"),
		filepath.Join("testdata", "strict", "persistence", "vo_vogue.go"),
		filepath.Join("testdata", "tab", "want", "vo_vogue.go"),
		filepath.Join("internal", "fixture", "vo_vogue.go"),
	}
	for _, golden := range goldens {
		t.Run(golden, func(t *testing.T) {
			// Arrange
			fset := token.NewFileSet()

			// Act
			file, err := parser.ParseFile(fset, golden, nil, 0)

			// Assert
			require.NoError(t, err)
			for _, decl := range file.Decls {
				switch decl := decl.(type) {
				case *ast.FuncDecl:
					if decl.Recv == nil {
						assert.True(t, strings.HasPrefix(decl.Name.Name, "New"),
							"%s: package-level function %s is not a constructor", fset.Position(decl.Pos()), decl.Name.Name)
					}
				case *ast.GenDecl:
					if decl.Tok == token.VAR {
						assertOnlyCompiledPatterns(t, fset, decl)
					}
				}
			}
		})
	}
}

// assertOnlyCompiledPatterns fails for a package-level variable that is not a
// regular expression compiled with regexp.MustCompile and named after its type
// with a "Pattern" suffix.
func assertOnlyCompiledPatterns(t *testing.T, fset *token.FileSet, decl *ast.GenDecl) {
	t.Helper()

	for _, spec := range decl.Specs {
		value, ok := spec.(*ast.ValueSpec)
		if !ok {
			continue
		}
		for i, name := range value.Names {
			assert.True(t, strings.HasSuffix(name.Name, "Pattern"),
				"%s: package-level variable %s is not a compiled pattern", fset.Position(name.Pos()), name.Name)
			require.Len(t, value.Values, len(value.Names))
			call, isCall := value.Values[i].(*ast.CallExpr)
			require.True(t, isCall, "%s: %s is not initialised by a call", fset.Position(name.Pos()), name.Name)
			assert.Equal(t, "regexp.MustCompile", types.ExprString(call.Fun),
				"%s: %s is not compiled by regexp.MustCompile", fset.Position(name.Pos()), name.Name)
		}
	}
}
