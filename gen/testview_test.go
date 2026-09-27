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
		assert.Contains(t, got, "func TestTitle(t *testing.T) {")
		assert.Contains(t, got, "voguetest.Scalar[Title, *Title, string]{")
		assert.Regexp(t, `Candidates:\s+\[\]string\{"a"\},`, got)
		assert.Contains(t, got, "t.Parallel()")
	})

	t.Run("merges the rules that reject the same input into one accumulation row", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title required min=1\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Regexp(t, `Rules:\s+\[\]string\{"required", "min"\},`, got)
	})

	t.Run("asserts a rejected input against every rule that rejects it", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title nodigits\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Regexp(t, `Field:\s+"title",`, got)
		assert.Regexp(t, `Name:\s+"rejects a value carrying a digit",\s+Input:\s+"a1",\s+Rules:\s+\[\]string\{"nodigits"\},`, got)
	})

	t.Run("hands the suite no candidate when the rules declare no example", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name: "opaque", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} is opaque",
			Emit: func(c vogue.EmitContext) string { return c.Var + ` != ""` },
		}))

		// Act
		got := generateTest(t, "//vogue:string Title opaque\n", set)

		// Assert
		assert.Regexp(t, `Candidates:\s+nil,`, got)
		assert.Regexp(t, `Rejected:\s+nil,`, got)
	})

	t.Run("ignores an example the kind cannot express", func(t *testing.T) {
		// Arrange
		body := "//vogue:int Covers min=1 max=200\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "voguetest.Scalar[Covers, *Covers, int64]{")
		assert.Regexp(t, `Candidates:\s+\[\]int64\{1, 200\},`, got)
		assert.NotContains(t, got, `"a"`)
	})

	t.Run("proves a normalizer rewrites instead of only accepting", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title trim required\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "Normalized: []voguetest.Normalization[string]{")
		assert.Contains(t, got, "Input: \"  a  \",\n\t\t\t\tOut:   \"a\",")
		assert.Regexp(t, `Get:\s+nil,`, got)
	})

	t.Run("asserts the value is held unchanged when no normalizer runs", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title required\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Regexp(t, `Get:\s+Title.String,`, got)
		assert.Regexp(t, `Normalized:\s+nil,`, got)
	})

	t.Run("pins the member set and order of an enum", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:enum TabStatus open,closed\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "voguetest.Enum[TabStatus, *TabStatus]{")
		assert.Contains(t, got, "All:   TabStatuses{}.All(),")
		assert.Contains(t, got, "Parse: TabStatuses{}.Parse,")
		assert.Contains(t, got, "Want: []string{\n\t\t\t\"open\",\n\t\t\t\"closed\",\n\t\t},")
	})

	t.Run("asserts the version an uuid identifier mints", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:id SessionID uuid4\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "voguetest.UUID[SessionID, *SessionID]{")
		assert.Contains(t, got, "FromString: NewSessionIDFromString,")
		assert.Contains(t, got, "Version:    4,")
	})

	t.Run("covers the positive rule of an int64 identifier", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:id InvoiceNumber int64\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "voguetest.Int64ID[InvoiceNumber, *InvoiceNumber]{")
		assert.Contains(t, got, "FromInt64:  NewInvoiceNumberFromInt64,")
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
		assert.Regexp(t, `Name:\s+"rejects \\"hi\\" \(shouty\)",`, got)
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
		assert.Contains(t, got, `Name:  "squash rewrites \" a \" to \"a\"",`)
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
		assert.Regexp(t, `Normalized:\s+nil,`, got)
		assert.Regexp(t, `Candidates:\s+\[\]int64\{3\},`, got)
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
		assert.Regexp(t, `Candidates:\s+\[\]string\{"clean"\},`, got)
		assert.Regexp(t, `Name:\s+"rejects \\"clash\\" \(second\)",`, got)
	})

	t.Run("generates a test file in the package under test", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:enum TabStatus open,closed\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "// Code generated by vogue. DO NOT EDIT.")
		assert.Contains(t, got, "package tab")
		assert.Contains(t, got, `"github.com/MathiasHilgert/vogue/voguetest"`)
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
