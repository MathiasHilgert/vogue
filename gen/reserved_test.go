package gen_test

import (
	"go/ast"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/rules"
)

// reservedNames are the identifiers the templates declare or import. A value
// object named after any of them must still compile: its receiver must not
// shadow what its methods use.
var reservedNames = []string{
	"data", "decimal", "driver", "err", "failed", "fmt", "id", "isNull", "member", "notification", "null",
	"number", "other", "parsed", "raw", "rawLength", "regexp", "schema", "slices",
	"source", "src", "strconv", "strings", "text", "textjson", "unicode", "utf8", "uuid", "validation", "value",
	"whole", "zero",
}

// TestGenerate_ReservedReceiverNames renders every kind of value object under
// every reserved name and type-checks the result.
func TestGenerate_ReservedReceiverNames(t *testing.T) {
	kinds := []string{
		"string %s trim required len=2 regex=^[a-z]+$ email url uuid timezone",
		"int %s min=1 max=9",
		"decimal %s min=0 max=1",
		"enum %s alpha,beta",
		"id %s",
		"id %s int64",
	}
	for _, kind := range kinds {
		t.Run(strings.Fields(kind)[0], func(t *testing.T) {
			// Arrange
			var body strings.Builder
			for _, name := range reservedNames {
				typeName := strings.ToUpper(name[:1]) + name[1:]
				body.WriteString("//vogue:" + strings.Replace(kind, "%s", typeName, 1) + "\n")
			}
			pkg := parseSource(t, body.String(), rules.MustSet())

			for _, sql := range []bool{false, true} {
				// Act
				files, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg, SQL: sql})
				require.NoError(t, err)

				// Assert
				fset := token.NewFileSet()
				code, err := parser.ParseFile(fset, "vo_vogue.go", files[0].Content, 0)
				require.NoError(t, err)
				config := types.Config{Importer: importer.ForCompiler(fset, "source", nil)}
				_, err = config.Check("tab", fset, []*ast.File{code}, nil)
				require.NoError(t, err, "a value object named after a reserved identifier does not compile")
			}
		})
	}
}
