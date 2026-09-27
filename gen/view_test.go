package gen_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/gen/internal/testrules"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generate renders one in-memory source body and returns the generated value
// objects, dropping the test file the generator produces alongside them.
func generate(t *testing.T, body string, rules *vogue.RuleSet) string {
	t.Helper()

	return generated(t, body, rules, gen.CodeFile)
}

// generated renders one in-memory source body and returns the generated file
// of the requested kind.
func generated(t *testing.T, body string, rules *vogue.RuleSet, kind gen.FileKind) string {
	t.Helper()

	files, err := filesOf(t, gen.Options{Package: parseSource(t, body, rules)})
	require.NoError(t, err)
	require.Len(t, files, 2)
	for _, file := range files {
		if file.Kind == kind {
			return string(file.Content)
		}
	}
	t.Fatalf("the generator produced no %s file", kind)
	return ""
}

func TestGenerator_Files_Kinds(t *testing.T) {
	t.Run("mints a random identifier for the uuid4 strategy", func(t *testing.T) {
		// Arrange
		body := "//vogue:id SessionID uuid4\n"

		// Act
		got := generate(t, body, &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "id, err := uuid.NewRandom()")
		assert.Contains(t, got, "a random UUIDv4")
	})

	t.Run("documents a directive that carries no doc comment", func(t *testing.T) {
		// Arrange
		body := "//vogue:enum Channel dine_in,takeaway\n"

		// Act
		got := generate(t, body, &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, `// Channel is the enum value object for the field "channel".`)
		assert.Contains(t, got, `func (Channels) DineIn() Channel { return Channel{value: "dine_in"} }`)
	})

	t.Run("parses a member with a switch over its members, not a loop over a fresh slice", func(t *testing.T) {
		// Act
		got := generate(t, "//vogue:enum Channel dine_in,takeaway\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "\tswitch raw {\n\tcase channels.DineIn().value:\n\t\treturn channels.DineIn(), nil\n")
		assert.NotContains(t, got, "range channels.All()")
	})

	t.Run("keeps a doc comment that already opens with the type name", func(t *testing.T) {
		// Arrange
		body := "// Channel is where an order came from.\n//vogue:enum Channel dine_in,takeaway\n"

		// Act
		got := generate(t, body, &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "// Channel is where an order came from.")
		assert.NotContains(t, got, "is the enum value object")
	})

	t.Run("emits a parameterised call through its versioned package", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name:    "prefix",
			Kinds:   vogue.Kinds(vogue.String),
			Message: "{{.Field}} must start with {{.Param}}",
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamString},
			Call:    &vogue.FuncRef{Path: "example.com/checks/v2", Name: "HasPrefix"},
		}))

		// Act
		got := generate(t, "//vogue:string Sku prefix=SKU-\n", set)

		// Assert
		assert.Contains(t, got, `"example.com/checks/v2"`)
		assert.Contains(t, got, `if !checks.HasPrefix(value, "SKU-") {`)
		assert.Regexp(t, `notification\.Reject\(\s*"sku",\s*"prefix",\s*"SKU\-",\s*value,\s*"sku\ must\ start\ with\ SKU\-",\s*\)`, got)
	})

	t.Run("leaves the SQL codec out when asked to", func(t *testing.T) {
		// Arrange
		pkg := parseSource(t, "//vogue:string Title required\n//vogue:enum Channel dine_in,takeaway\n", testrules.Set(testrulesPath))

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg, OmitSQL: true})

		// Assert
		require.NoError(t, err)
		code := string(files[0].Content)
		assert.NotContains(t, code, "database/sql/driver")
		assert.NotContains(t, code, "Scan(")
		assert.NotContains(t, code, "Value()")
		assert.NotContains(t, code, "nolint:recvcheck", "without Scan every method has a value receiver but UnmarshalText")
	})

	t.Run("generates nothing for a file without directives", func(t *testing.T) {
		// Arrange
		pkg := parseSource(t, "// nothing to generate here\n", &vogue.RuleSet{})

		// Act
		files, err := filesOf(t, gen.Options{Package: pkg})

		// Assert
		require.NoError(t, err)
		assert.Empty(t, files)
	})

	t.Run("reports a message template that cannot be rendered", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name:    "odd",
			Kinds:   vogue.Kinds(vogue.String),
			Message: "{{.Missing}}",
			Emit:    func(c vogue.EmitContext) string { return c.Var + ` != ""` },
		}))

		// Act
		_, err := filesOf(t, gen.Options{Package: parseSource(t, "//vogue:string Title odd\n", set)})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Missing")
	})

	t.Run("rejects a package without a name", func(t *testing.T) {
		// Act
		g, err := gen.New(gen.Options{Package: &parse.Package{}})

		// Assert
		require.Error(t, err)
		assert.Nil(t, g)
		assert.Contains(t, err.Error(), "package name must not be empty")
	})
}

func TestGenerator_Files_ImportCollisions(t *testing.T) {
	t.Run("reports a rule import that shadows one the kind already needs", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name:    "shadow",
			Kinds:   vogue.Kinds(vogue.String),
			Message: "{{.Field}} is invalid",
			Imports: []string{"example.com/mine/fmt"},
			Emit:    func(c vogue.EmitContext) string { return "fmt.Valid(" + c.Var + ")" },
		}))

		// Act
		_, err := filesOf(t, gen.Options{Package: parseSource(t, "//vogue:string Title shadow\n", set)})

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "import collision")
		assert.Contains(t, err.Error(), "example.com/mine/fmt")
	})
}
