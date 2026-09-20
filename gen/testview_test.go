package gen_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/gen"
	"github.com/MathiasHilgert/vogue/gen/internal/testrules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTest renders one in-memory source body and returns the generated
// test file.
func generateTest(t *testing.T, body string, rules *vogue.RuleSet) string {
	t.Helper()

	return generated(t, body, rules, gen.TestFile)
}

func TestGenerator_Tests(t *testing.T) {
	t.Run("derives the accepted row from a valid example no other rule rejects", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title required min=1\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "func TestNewTitle(t *testing.T) {")
		assert.Contains(t, got, `{name: "accepts \"a\"", in: "a"},`)
		assert.Contains(t, got, "// Arrange")
		assert.Contains(t, got, "t.Parallel()")
	})

	t.Run("merges the rules that reject the same input into one accumulation row", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title required min=1\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, `wantRules: []string{"required", "min"}`)
	})

	t.Run("asserts a rejected input against every rule that rejects it", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title nodigits\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, `assert.ErrorIs(t, err, vogue.FieldError{Field: "title", Rule: rule}`)
		assert.Contains(t, got, `{name: "rejects a value carrying a digit", in: "a1", wantRules: []string{"nodigits"}},`)
	})

	t.Run("skips a constructor whose rules declare no usable example", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name: "opaque", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} is opaque",
			Emit: func(c vogue.EmitContext) string { return c.Var + ` != ""` },
		}))

		// Act
		got := generateTest(t, "//vogue:string Title opaque\n", set)

		// Assert
		assert.Contains(t, got, "t.Skip(")
		assert.Contains(t, got, "add Examples to the rules it uses")
		assert.NotContains(t, got, "func TestTitle_IsZero")
	})

	t.Run("ignores an example the kind cannot express", func(t *testing.T) {
		// Arrange
		body := "//vogue:int Covers min=1 max=200\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "in        int64")
		assert.Contains(t, got, `{name: "accepts \"1\"", in: 1},`)
		assert.NotContains(t, got, `in: "a"`)
	})

	t.Run("proves a normalizer rewrites instead of only accepting", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title trim required\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "func TestNewTitle_Normalizes(t *testing.T) {")
		assert.Contains(t, got, `in: "  a  ", want: "a"`)
		assert.NotContains(t, got, "assert.Equal(t, tc.in, got.String())")
	})

	t.Run("asserts the value is held unchanged when no normalizer runs", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title required\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "assert.Equal(t, tc.in, got.String())")
		assert.NotContains(t, got, "func TestNewTitle_Normalizes")
	})

	t.Run("pins the member set and order of an enum", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:enum TabStatus open,closed\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "func TestTabStatusValues(t *testing.T) {")
		assert.Contains(t, got, `want := []string{"open", "closed"}`)
		assert.Contains(t, got, `Rule: "oneof", Param: "open,closed"`)
	})

	t.Run("asserts the version an uuid identifier mints", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:id SessionID uuid4\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "assert.EqualValues(t, 4, first.Version())")
		assert.Contains(t, got, `assert.ErrorIs(t, err, vogue.FieldError{Field: "sessionId", Rule: "uuid"})`)
	})

	t.Run("covers the positive rule of an int64 identifier", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:id InvoiceNumber int64\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "func TestInvoiceNumberFromInt64(t *testing.T) {")
		assert.Contains(t, got, `Rule: "positive"`)
		assert.NotContains(t, got, "func TestNewInvoiceNumber")
	})

	t.Run("names a row after its input when the example carries no note", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name: "shouty", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} must shout",
			Emit: func(c vogue.EmitContext) string { return c.Var + ` != ""` },
			Examples: vogue.Examples{
				Valid:   []vogue.Example{{In: "HI"}},
				Invalid: []vogue.Example{{In: "hi"}},
			},
		}))

		// Act
		got := generateTest(t, "//vogue:string Title shouty\n", set)

		// Assert
		assert.Contains(t, got, `{name: "rejects \"hi\" (shouty)", in: "hi", wantRules: []string{"shouty"}},`)
	})

	t.Run("names a rewrite after its rule when the normalization carries no note", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(
			vogue.Rule{
				Name: "squash", Kinds: vogue.Kinds(vogue.String, vogue.Int), Message: "{{.Field}} is squashed",
				Normalize: true, Imports: []string{"strings"},
				Emit:     func(c vogue.EmitContext) string { return c.Var + " = strings.TrimSpace(" + c.Var + ")" },
				Examples: vogue.Examples{Normalized: []vogue.Normalization{{In: " a ", Out: "a"}}},
			},
			vogue.Rule{
				Name: "any", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} is anything",
				Emit:     func(c vogue.EmitContext) string { return c.Var + ` != "\x00"` },
				Examples: vogue.Examples{Valid: []vogue.Example{{In: "a"}}},
			},
		))

		// Act
		got := generateTest(t, "//vogue:string Title squash any\n", set)

		// Assert
		assert.Contains(t, got, `name: "squash rewrites \" a \" to \"a\""`)
	})

	t.Run("drops a rewrite the kind cannot express", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(
			vogue.Rule{
				Name: "squash", Kinds: vogue.Kinds(vogue.Int), Message: "{{.Field}} is squashed",
				Normalize: true,
				Emit:      func(c vogue.EmitContext) string { return c.Var + " = " + c.Var },
				Examples:  vogue.Examples{Normalized: []vogue.Normalization{{In: " a ", Out: "a"}}},
			},
			vogue.Rule{
				Name: "any", Kinds: vogue.Kinds(vogue.Int), Message: "{{.Field}} is anything",
				Emit:     func(c vogue.EmitContext) string { return c.Var + " == " + c.Var },
				Examples: vogue.Examples{Valid: []vogue.Example{{In: "3"}}},
			},
		))

		// Act
		got := generateTest(t, "//vogue:int Covers squash any\n", set)

		// Assert
		assert.NotContains(t, got, "_Normalizes")
		assert.Contains(t, got, "{name: \"accepts \\\"3\\\"\", in: 3},")
	})

	t.Run("refuses a sample another rule of the same directive rejects", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(
			vogue.Rule{
				Name: "first", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} fails first",
				Emit:     func(c vogue.EmitContext) string { return c.Var + ` != "" ` },
				Examples: vogue.Examples{Valid: []vogue.Example{{In: "clash"}, {In: "clean"}}},
			},
			vogue.Rule{
				Name: "second", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} fails second",
				Emit:     func(c vogue.EmitContext) string { return c.Var + ` != "clash"` },
				Examples: vogue.Examples{Invalid: []vogue.Example{{In: "clash"}}},
			},
		))

		// Act
		got := generateTest(t, "//vogue:string Title first second\n", set)

		// Assert
		assert.Contains(t, got, `{name: "accepts \"clean\"", in: "clean"},`)
		assert.Contains(t, got, `{name: "rejects \"clash\" (second)", in: "clash", wantRules: []string{"second"}},`)
	})

	t.Run("generates a test file in the package under test", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:enum TabStatus open,closed\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "// Code generated by vogue. DO NOT EDIT.")
		assert.Contains(t, got, "package tab")
		assert.Contains(t, got, `"github.com/stretchr/testify/require"`)
	})
}

func TestFileKind_String(t *testing.T) {
	t.Run("names every kind", func(t *testing.T) {
		// Act & Assert
		assert.Equal(t, "code", gen.CodeFile.String())
		assert.Equal(t, "test", gen.TestFile.String())
		assert.Equal(t, "FileKind(9)", gen.FileKind(9).String())
	})
}
