package generator_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/generator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// copyDir copies a testdata package into a fresh temporary directory, so a run
// that writes files never touches the committed fixture.
func copyDir(t *testing.T, src string) string {
	t.Helper()

	dst := t.TempDir()
	entries, err := os.ReadDir(src)
	require.NoError(t, err)
	for _, entry := range entries {
		require.False(t, entry.IsDir(), "the fixture is expected to be flat")
		content, err := os.ReadFile(filepath.Join(src, entry.Name()))
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dst, entry.Name()), content, 0o600))
	}
	return dst
}

// names returns the file names present in a directory.
func names(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name())
	}
	return out
}

// cuit is a custom rule standing in for the ones a project brings of its own.
var cuit = vogue.Rule{
	Name:    "elevendigits",
	Kinds:   vogue.Kinds(vogue.String),
	Doc:     "Requires exactly eleven decimal digits.",
	Message: "{{.Field}} must be eleven digits",
	Imports: []string{"regexp"},
	Declare: func(vogue.EmitContext) string {
		return "var _elevenDigits = regexp.MustCompile(`^[0-9]{11}$`)"
	},
	Emit: func(c vogue.EmitContext) string { return "_elevenDigits.MatchString(" + c.Var + ")" },
	Examples: vogue.Examples{
		Valid:   []vogue.Example{{In: "20123456789", Note: "eleven digits"}},
		Invalid: []vogue.Example{{In: "123", Note: "too few digits"}},
	},
}

func TestRun(t *testing.T) {
	t.Run("writes a value object and its test next to every source file", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		var stdout, stderr bytes.Buffer

		// Act
		err := generator.Run(
			generator.WithDir(dir),
			generator.WithImportPath("example.test/tab"),
			generator.WithStdout(&stdout),
			generator.WithStderr(&stderr),
		)

		// Assert
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"vo.go", "vo_vogue.go", "vo_vogue_test.go"}, names(t, dir))
		assert.Empty(t, stderr.String())

		generated, err := os.ReadFile(filepath.Join(dir, "vo_vogue.go"))
		require.NoError(t, err)
		assert.Contains(t, string(generated), "func NewTitle(raw string) (Title, error)")
	})

	t.Run("skips the generated tests when they are turned off", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))

		// Act
		err := generator.Run(generator.WithDir(dir), generator.WithTests(false))

		// Assert
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"vo.go", "vo_vogue.go"}, names(t, dir))
	})

	t.Run("leaves the SQL codec out when it is turned off", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))

		// Act
		err := generator.Run(generator.WithDir(dir), generator.WithSQL(false))

		// Assert
		require.NoError(t, err)
		generated, err := os.ReadFile(filepath.Join(dir, "vo_vogue.go"))
		require.NoError(t, err)
		assert.NotContains(t, string(generated), "database/sql/driver")
	})

	t.Run("adds the JSONSchema method when asked to", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))

		// Act
		err := generator.Run(generator.WithDir(dir), generator.WithSchema(true))

		// Assert
		require.NoError(t, err)
		generated, err := os.ReadFile(filepath.Join(dir, "vo_vogue.go"))
		require.NoError(t, err)
		assert.Contains(t, string(generated), "JSONSchema() schema.Schema")
	})

	t.Run("a dry run reports what it would write and writes nothing", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		var stdout bytes.Buffer

		// Act
		err := generator.Run(generator.WithDir(dir), generator.WithDryRun(true), generator.WithStdout(&stdout))

		// Assert
		require.NoError(t, err)
		assert.ElementsMatch(t, []string{"vo.go"}, names(t, dir))
		assert.Contains(t, stdout.String(), "vo_vogue.go")
		assert.Contains(t, stdout.String(), "vo_vogue_test.go")
		assert.Contains(t, stdout.String(), "bytes")
	})

	t.Run("reports every diagnostic on stderr and fails", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "broken"))
		var stderr bytes.Buffer

		// Act
		err := generator.Run(generator.WithDir(dir), generator.WithStderr(&stderr))

		// Assert
		require.Error(t, err)
		require.EqualError(t, err, "vogue: 2 errors")
		assert.Contains(t, stderr.String(), `unknown rule "mni"`)
		assert.Contains(t, stderr.String(), `did you mean "min"?`)
		assert.Contains(t, stderr.String(), `rule "email" does not apply to kind int`)
		assert.Equal(t, 2, bytes.Count(stderr.Bytes(), []byte("\n")), "one diagnostic per line")
	})

	t.Run("a custom rule is usable in a directive", func(t *testing.T) {
		// Arrange
		dir := t.TempDir()
		source := "package tax\n\n// TaxID identifies a taxpayer.\n//vogue:string TaxID trim elevendigits\n"
		require.NoError(t, os.WriteFile(filepath.Join(dir, "vo.go"), []byte(source), 0o600))

		// Act
		err := generator.Run(generator.WithDir(dir), generator.WithRules(cuit))

		// Assert
		require.NoError(t, err)
		generated, err := os.ReadFile(filepath.Join(dir, "vo_vogue.go"))
		require.NoError(t, err)
		assert.Contains(t, string(generated), "_elevenDigits.MatchString")
		assert.Contains(t, string(generated), "strings.TrimSpace", "the built-ins are still available")
	})

	t.Run("a custom rule shadowing a built-in names both", func(t *testing.T) {
		// Arrange
		shadow := cuit
		shadow.Name = "trim"

		// Act
		err := generator.Run(generator.WithDir(t.TempDir()), generator.WithRules(shadow))

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `custom rule "trim"`)
		assert.Contains(t, err.Error(), `built-in rule "trim"`)
		assert.Contains(t, err.Error(), "WithoutBuiltins")
	})

	t.Run("without the built-ins only the custom rules are known", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		var stderr bytes.Buffer

		// Act
		err := generator.Run(generator.WithDir(dir), generator.WithoutBuiltins(), generator.WithStderr(&stderr))

		// Assert
		require.Error(t, err)
		assert.Contains(t, stderr.String(), `unknown rule "trim"`)
	})

	t.Run("defaults to the directory go generate is running in", func(t *testing.T) {
		// Arrange
		dir := copyDir(t, filepath.Join("testdata", "tab"))
		t.Setenv("GOFILE", filepath.Join(dir, "vo.go"))

		// Act
		err := generator.Run()

		// Assert
		require.NoError(t, err)
		assert.Contains(t, names(t, dir), "vo_vogue.go")
	})

	t.Run("refuses a directory that does not exist", func(t *testing.T) {
		// Act
		err := generator.Run(generator.WithDir(filepath.Join(t.TempDir(), "absent")))

		// Assert
		require.Error(t, err)
	})
}

// predicateRule returns a rule dispatching to a static function of the given
// package, the shape a consumer-owned predicate has.
func predicateRule(path, name string) vogue.Rule {
	return vogue.Rule{
		Name:    "predicate",
		Kinds:   vogue.Kinds(vogue.String),
		Doc:     "Requires what the predicate accepts.",
		Message: "{{.Field}} is not acceptable",
		Call:    &vogue.FuncRef{Path: path, Name: name},
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{In: "abc"}},
			Invalid: []vogue.Example{{In: ""}},
		},
	}
}

func TestRun_CallRules(t *testing.T) {
	const module = "github.com/MathiasHilgert/vogue"

	t.Run("refuses a call into a package that depends on vogue", func(t *testing.T) {
		// Arrange
		var stdout bytes.Buffer

		// Act
		err := generator.Run(
			generator.WithDir(filepath.Join("testdata", "callrule")),
			generator.WithRules(predicateRule(module+"/rules", "Any")),
			generator.WithDryRun(true),
			generator.WithStdout(&stdout),
		)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `rule "predicate"`)
		assert.Contains(t, err.Error(), module+"/rules")
		assert.Contains(t, err.Error(), "depends on vogue")
		assert.Empty(t, stdout.String(), "nothing is planned for a refused run")
	})

	t.Run("refuses a call into the vogue module itself", func(t *testing.T) {
		// Act
		err := generator.Run(
			generator.WithDir(filepath.Join("testdata", "callrule")),
			generator.WithRules(predicateRule(module, "Any")),
			generator.WithDryRun(true),
		)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "depends on vogue")
	})

	t.Run("accepts a call into a package that does not depend on vogue", func(t *testing.T) {
		// Act
		err := generator.Run(
			generator.WithDir(filepath.Join("testdata", "callrule")),
			generator.WithRules(predicateRule("unicode/utf8", "ValidString")),
			generator.WithDryRun(true),
			generator.WithStdout(io.Discard),
		)

		// Assert
		require.NoError(t, err)
	})

	t.Run("warns, and generates, when the dependencies cannot be listed", func(t *testing.T) {
		// Arrange
		var stderr bytes.Buffer

		// Act
		err := generator.Run(
			generator.WithDir(filepath.Join("testdata", "callrule")),
			generator.WithRules(predicateRule("example.invalid/nowhere", "Check")),
			generator.WithDryRun(true),
			generator.WithStdout(io.Discard),
			generator.WithStderr(&stderr),
		)

		// Assert
		require.NoError(t, err)
		assert.Contains(t, stderr.String(), "example.invalid/nowhere")
		assert.Contains(t, stderr.String(), "could not be checked")
	})
}
