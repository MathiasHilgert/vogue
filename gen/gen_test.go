package gen_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/gen/internal/testrules"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// update rewrites the golden files instead of comparing against them.
var update = flag.Bool("update", false, "rewrite the generator golden files")

// testrulesPath is the import path the call-backed test rule dispatches to.
const testrulesPath = "github.com/MathiasHilgert/vogue/gen/internal/testrules"

// parseCase parses one golden case directory with the test catalogue.
func parseCase(t *testing.T, name string) *parse.Package {
	t.Helper()

	pkg, err := parse.Dir(filepath.Join("testdata", name), testrules.Set(testrulesPath))
	require.NoError(t, err)
	return pkg
}

// assertGolden compares generated content against the committed golden file of
// the tab case, rewriting it first when -update is set.
func assertGolden(t *testing.T, name string, content []byte) {
	t.Helper()

	golden := filepath.Join("testdata", "tab", "want", name)
	if *update {
		require.NoError(t, os.WriteFile(golden, content, 0o600))
	}
	want, err := os.ReadFile(golden)
	require.NoError(t, err)
	assert.Equal(t, string(want), string(content))
}

func TestNew(t *testing.T) {
	t.Run("rejects a missing package", func(t *testing.T) {
		// Arrange
		opts := gen.Options{Validation: testValidation}

		// Act
		g, err := gen.New(opts)

		// Assert
		require.Error(t, err)
		assert.Nil(t, g)
		assert.Contains(t, err.Error(), "package must not be nil")
	})

	t.Run("accepts a parsed package", func(t *testing.T) {
		// Arrange
		opts := gen.Options{Validation: testValidation, Package: parseCase(t, "tab")}

		// Act
		g, err := gen.New(opts)

		// Assert
		require.NoError(t, err)
		assert.NotNil(t, g)
	})
}

func TestGenerator_Files(t *testing.T) {
	t.Run("generates one file per source file, matching the golden output", func(t *testing.T) {
		// Arrange
		g, err := gen.New(gen.Options{Validation: testValidation, Package: parseCase(t, "tab")})
		require.NoError(t, err)

		// Act
		files, err := g.Files()

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 2)
		assert.Equal(t, gen.CodeFile, files[0].Kind)
		assert.Equal(t, filepath.Join("testdata", "tab", "vo_vogue.go"), files[0].Path)
		assertGolden(t, "vo_vogue.go", files[0].Content)
	})

	t.Run("generates a test file next to every generated file", func(t *testing.T) {
		// Arrange
		g, err := gen.New(gen.Options{Validation: testValidation, Package: parseCase(t, "tab")})
		require.NoError(t, err)

		// Act
		files, err := g.Files()

		// Assert
		require.NoError(t, err)
		require.Len(t, files, 2)
		assert.Equal(t, gen.TestFile, files[1].Kind)
		assert.Equal(t, filepath.Join("testdata", "tab", "vo_vogue_test.go"), files[1].Path)
		assertGolden(t, "vo_vogue_test.go", files[1].Content)
	})

	t.Run("rejects a message template that references the runtime value", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name:    "shouty",
			Kinds:   vogue.Kinds(vogue.String),
			Message: "{{.Field}} rejects {{.Value}}",
			Emit:    func(c vogue.EmitContext) string { return c.Var + ` != ""` },
		}))
		pkg := parseSource(t, "//vogue:string Title shouty\n", set)

		// Act
		_, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), ".Value")
		assert.Contains(t, err.Error(), "shouty")
	})

	t.Run("rejects two rules whose imports collide on the same identifier", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(
			vogue.Rule{
				Name: "alpha", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} is not alpha",
				Call: &vogue.FuncRef{Path: "example.com/a/check", Name: "IsAlpha"},
			},
			vogue.Rule{
				Name: "beta", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} is not beta",
				Call: &vogue.FuncRef{Path: "example.com/b/check", Name: "IsBeta"},
			},
		))
		pkg := parseSource(t, "//vogue:string Title alpha beta\n", set)

		// Act
		_, err := filesOf(t, gen.Options{Validation: testValidation, Package: pkg})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "import collision")
		assert.Contains(t, err.Error(), "check")
	})
}

func TestWrite(t *testing.T) {
	t.Run("writes every file", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		files := []gen.OutFile{{Path: filepath.Join(dir, "vo_vogue.go"), Content: []byte("package x\n")}}

		// Act
		err := gen.Write(files)

		// Assert
		require.NoError(t, err)
		got, err := os.ReadFile(files[0].Path)
		require.NoError(t, err)
		assert.Equal(t, "package x\n", string(got))
	})

	t.Run("reports an unwritable destination", func(t *testing.T) {
		// Arrange
		files := []gen.OutFile{{Path: filepath.Join(t.TempDir(), "missing", "vo_vogue.go")}}

		// Act
		err := gen.Write(files)

		// Assert
		require.Error(t, err)
	})
}

func TestWrite_Rename(t *testing.T) {
	t.Run("reports a destination that cannot be replaced", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		occupied := filepath.Join(dir, "vo_vogue.go")
		require.NoError(t, os.Mkdir(occupied, 0o750))

		// Act
		err := gen.Write([]gen.OutFile{{Path: occupied, Content: []byte("package x\n")}})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), occupied, "a destination that cannot be read is reported before anything is written")
	})
}
