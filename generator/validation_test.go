package generator_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/MathiasHilgert/vogue/generator"
)

func TestParseValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		spec     string
		wantPath string
		wantType string
		wantErr  string
	}{
		{name: "a path with dots in its host", spec: "example.com/app/fault.Validation", wantPath: "example.com/app/fault", wantType: "Validation"},
		{name: "a path of one element", spec: "fault.Validation", wantPath: "fault", wantType: "Validation"},
		{name: "a versioned path", spec: "example.com/app/v2/fault.Failures", wantPath: "example.com/app/v2/fault", wantType: "Failures"},
		{name: "no type", spec: "example.com/app/fault", wantErr: "must be <import path>.<Type>"},
		{name: "a dot only in the host", spec: "example.com", wantErr: "must be exported"},
		{name: "an empty path", spec: ".Validation", wantErr: "must be <import path>.<Type>"},
		{name: "an empty type", spec: "example.com/fault.", wantErr: "must be <import path>.<Type>"},
		{name: "an unexported type", spec: "example.com/fault.validation", wantErr: "must be exported"},
		{name: "nothing", spec: "", wantErr: "must be <import path>.<Type>"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			// Act
			path, typeName, err := generator.ParseValidation(test.spec)

			// Assert
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)

				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.wantPath, path)
			assert.Equal(t, test.wantType, typeName)
		})
	}
}

func TestRun_RequiresTheFailureType(t *testing.T) {
	t.Parallel()

	// Arrange
	dir := copyDir(t, filepath.Join("testdata", "tab"))

	// Act
	err := generator.Run(generator.WithDir(dir))

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "-validation=<import path>.<Type>")
	assert.Contains(t, err.Error(), "Add(field, rule, message string)")
	assert.Equal(t, []string{"vo.go"}, names(t, dir), "nothing is written without it")
}

func TestValidationReference(t *testing.T) {
	t.Parallel()

	// declared returns the functions and types a file declares, by name.
	declared := func(t *testing.T, source any) []string {
		t.Helper()

		file, err := parser.ParseFile(token.NewFileSet(), "reference.go", source, 0)
		require.NoError(t, err)

		var out []string
		for _, decl := range file.Decls {
			switch decl := decl.(type) {
			case *ast.FuncDecl:
				out = append(out, decl.Name.Name)
			case *ast.GenDecl:
				for _, spec := range decl.Specs {
					if typeSpec, ok := spec.(*ast.TypeSpec); ok {
						out = append(out, typeSpec.Name.Name)
					}
				}
			}
		}

		return out
	}

	t.Run("is valid Go that declares the contract", func(t *testing.T) {
		t.Parallel()

		// Act
		got := declared(t, generator.ValidationReference)

		// Assert
		assert.Subset(t, got, []string{"Validation", "Add", "Err", "Error", "Has"})
	})

	t.Run("declares what the example it condenses declares", func(t *testing.T) {
		t.Parallel()

		// Arrange
		example, err := os.ReadFile(filepath.Join("..", "examples", "validation", "validation.go"))
		require.NoError(t, err)

		// Act
		got := declared(t, generator.ValidationReference)

		// Assert
		assert.ElementsMatch(t, declared(t, string(example)), got)
	})

	t.Run("is what the command prints without the flag", func(t *testing.T) {
		t.Parallel()

		// Assert
		assert.True(t, strings.HasSuffix(generator.ValidationRequired, generator.ValidationReference))
	})
}
