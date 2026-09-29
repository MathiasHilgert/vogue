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
		assert.Contains(t, got, "func TestTitle_RoundTripsText(t *testing.T) {")
		assert.Contains(t, got, `for _, input := range []string{"a"} {`)
		assert.Contains(t, got, "t.Parallel()")
	})

	t.Run("stops a row at the precondition that rejects it", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title required min=1\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, `[]string{"required"}`,
			"required is a precondition, so the constructor stops after it and min never reports the empty string")
		assert.NotContains(t, got, `[]string{"required", "min"}`)
	})

	t.Run("asserts a rejected input against every rule that rejects it", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title nodigits\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, `failed.Has("title", rule)`)
		assert.Contains(t, got, `{"rejects a value carrying a digit", "a1", []string{"nodigits"}}`)
	})

	t.Run("writes no table when the rules declare no example", func(t *testing.T) {
		// Arrange
		set := &vogue.RuleSet{}
		require.NoError(t, set.Add(vogue.Rule{
			Name: "opaque", Kinds: vogue.Kinds(vogue.String), Message: "{{.Field}} is opaque",
			Emit: func(c vogue.EmitContext) string { return c.Var + ` != ""` },
		}))

		// Act
		got := generateTest(t, "//vogue:string Title opaque\n", set)

		// Assert
		assert.NotContains(t, got, "RoundTripsText")
		assert.NotContains(t, got, "RejectsInvalidInput")
		assert.Contains(t, got, "TestTitle_ZeroHasNoText")
	})

	t.Run("ignores an example the kind cannot express", func(t *testing.T) {
		// Arrange
		body := "//vogue:int Covers min=1 max=200\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "[]int64{1, 200}")
		assert.NotContains(t, got, `"a"`)
	})

	t.Run("proves a normalizer rewrites instead of only accepting", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title trim required\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.Contains(t, got, "TestTitle_Normalizes")
		assert.Contains(t, got, `"  a  ", "a"`)
		assert.NotContains(t, got, "AcceptsItsExamples")
	})

	t.Run("asserts the value is held unchanged when no normalizer runs", func(t *testing.T) {
		// Arrange
		body := "//vogue:string Title required\n"

		// Act
		got := generateTest(t, body, testrules.Set(testrulesPath))

		// Assert
		assert.NotContains(t, got, "Normalizes")
	})

	t.Run("pins the member set and order of an enum", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:enum TabStatus open,closed\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "func TestTabStatus_ListsItsMembers(t *testing.T) {")
		assert.Contains(t, got, `want := []string{"open", "closed"}`)
		assert.Contains(t, got, "TabStatuses{}.Parse(raw)")
	})

	t.Run("asserts the version an uuid identifier mints", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:id SessionID uuid4\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "func TestSessionID_MintsFreshIdentifiers(t *testing.T) {")
		assert.Contains(t, got, "if version != 4 {")
	})

	t.Run("covers the positive rule of an int64 identifier", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:id InvoiceNumber int64\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "func TestInvoiceNumber_RejectsWhatNoSequenceHandsOut(t *testing.T) {")
		assert.Contains(t, got, `failed.Has("invoiceNumber", `)
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
		assert.Contains(t, got, `{"rejects \"hi\" (shouty)", "hi", []string{"shouty"}}`)
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
		assert.Contains(t, got, `{"squash rewrites \" a \" to \"a\"", " a ", "a"}`)
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
		assert.NotContains(t, got, "Normalizes")
		assert.Contains(t, got, "[]int64{3}")
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
		assert.Contains(t, got, `[]string{"clean"}`)
		assert.Contains(t, got, `{"rejects \"clash\" (second)", "clash", []string{"second"}}`)
	})

	t.Run("generates a test file in the package under test", func(t *testing.T) {
		// Act
		got := generateTest(t, "//vogue:enum TabStatus open,closed\n", &vogue.RuleSet{})

		// Assert
		assert.Contains(t, got, "// Code generated by vogue. DO NOT EDIT.")
		assert.Contains(t, got, "package tab")
		assert.NotContains(t, got, "vogue/", "a generated test imports nothing of vogue")
		assert.NotContains(t, got, "testify")
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
