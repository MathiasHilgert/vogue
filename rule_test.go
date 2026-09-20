package vogue_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// minRule is a representative emit-based rule used across the rule tests.
func minRule() vogue.Rule {
	return vogue.Rule{
		Name:    "min",
		Kinds:   vogue.Kinds(vogue.String, vogue.Int),
		Doc:     "Rejects values shorter than the parameter.",
		Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
		Message: "{{.Field}} must be at least {{.Param}} characters",
		Emit:    func(c vogue.EmitContext) string { return "len(" + c.Var + ") >= " + c.Param },
		Examples: vogue.Examples{
			Valid:   []vogue.Example{{Param: "1", In: "a"}},
			Invalid: []vogue.Example{{Param: "1", In: ""}},
		},
	}
}

// emailRule is a representative call-based rule with no parameter.
func emailRule() vogue.Rule {
	return vogue.Rule{
		Name:    "email",
		Kinds:   vogue.Kinds(vogue.String),
		Doc:     "Rejects values that are not email addresses.",
		Message: "{{.Field}} must be a valid email address",
		Call:    &vogue.FuncRef{Path: "github.com/MathiasHilgert/vogue/rules", Name: "IsEmail"},
	}
}

func TestRule_Validate(t *testing.T) {
	t.Run("accepts a well formed rule", func(t *testing.T) {
		// Arrange
		cases := map[string]vogue.Rule{"emit": minRule(), "call": emailRule()}

		for name, rule := range cases {
			t.Run(name, func(t *testing.T) {
				// Act
				err := rule.Validate()

				// Assert
				assert.NoError(t, err)
			})
		}
	})

	// Arrange
	cases := []struct {
		name    string
		mutate  func(*vogue.Rule)
		wantMsg string
	}{
		{name: "empty name", mutate: func(r *vogue.Rule) { r.Name = "" }, wantMsg: "name must not be empty"},
		{name: "name with uppercase", mutate: func(r *vogue.Rule) { r.Name = "Min" }, wantMsg: "is not a valid tag name"},
		{name: "name starting with digit", mutate: func(r *vogue.Rule) { r.Name = "1min" }, wantMsg: "is not a valid tag name"},
		{name: "name with dash", mutate: func(r *vogue.Rule) { r.Name = "min-len" }, wantMsg: "is not a valid tag name"},
		{name: "no kinds", mutate: func(r *vogue.Rule) { r.Kinds = 0 }, wantMsg: "must declare at least one kind"},
		{name: "no message", mutate: func(r *vogue.Rule) { r.Message = "" }, wantMsg: "message must not be empty"},
		{name: "invalid message template", mutate: func(r *vogue.Rule) { r.Message = "{{.Field" }, wantMsg: "message template"},
		{
			name:    "neither emit nor call",
			mutate:  func(r *vogue.Rule) { r.Emit = nil; r.Call = nil },
			wantMsg: "exactly one of Emit or Call must be set",
		},
		{
			name: "both emit and call",
			mutate: func(r *vogue.Rule) {
				r.Call = &vogue.FuncRef{Path: "p", Name: "N"}
			},
			wantMsg: "exactly one of Emit or Call must be set",
		},
		{
			name:    "call without path",
			mutate:  func(r *vogue.Rule) { r.Emit = nil; r.Call = &vogue.FuncRef{Name: "IsEmail"} },
			wantMsg: "call path must not be empty",
		},
		{
			name:    "call without name",
			mutate:  func(r *vogue.Rule) { r.Emit = nil; r.Call = &vogue.FuncRef{Path: "p"} },
			wantMsg: "call name must not be empty",
		},
		{
			name:    "param type without presence",
			mutate:  func(r *vogue.Rule) { r.Param = vogue.ParamSpec{Presence: vogue.ParamNone, Type: vogue.ParamInt} },
			wantMsg: "param type must not be set when the rule takes no parameter",
		},
		{
			name:    "presence without type",
			mutate:  func(r *vogue.Rule) { r.Param = vogue.ParamSpec{Presence: vogue.ParamRequired} },
			wantMsg: "param type must be set",
		},
		{
			name:    "unknown presence",
			mutate:  func(r *vogue.Rule) { r.Param = vogue.ParamSpec{Presence: vogue.ParamPresence(9), Type: vogue.ParamInt} },
			wantMsg: "unknown param presence",
		},
		{
			name: "unknown param type",
			mutate: func(r *vogue.Rule) {
				r.Param = vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamType(9)}
			},
			wantMsg: "unknown param type",
		},
		{
			name: "declare without emit",
			mutate: func(r *vogue.Rule) {
				r.Emit = nil
				r.Call = &vogue.FuncRef{Path: "p", Name: "N"}
				r.Declare = func(vogue.EmitContext) string { return "var x = 1" }
			},
			wantMsg: "Declare requires Emit",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			rule := minRule()
			tc.mutate(&rule)

			// Act
			err := rule.Validate()

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantMsg)
			assert.Contains(t, err.Error(), "vogue:")
		})
	}
}

func TestFuncRef_Import(t *testing.T) {
	// Arrange
	ref := vogue.FuncRef{Path: "github.com/MathiasHilgert/vogue/rules", Name: "IsEmail"}

	// Act
	got := ref.Import()

	// Assert
	assert.Equal(t, "github.com/MathiasHilgert/vogue/rules", got)
}

func TestFuncRef_Selector(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		ref  vogue.FuncRef
		want string
	}{
		{name: "nested path", ref: vogue.FuncRef{Path: "a/b/rules", Name: "IsEmail"}, want: "rules.IsEmail"},
		{name: "single element path", ref: vogue.FuncRef{Path: "strings", Name: "HasPrefix"}, want: "strings.HasPrefix"},
		{name: "versioned path", ref: vogue.FuncRef{Path: "example.com/x/v2", Name: "F"}, want: "x.F"},
		{name: "empty path", ref: vogue.FuncRef{Name: "F"}, want: "F"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := tc.ref.Selector()

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRuleSet_Add(t *testing.T) {
	t.Run("adds rules in order", func(t *testing.T) {
		// Arrange
		var set vogue.RuleSet

		// Act
		err := set.Add(minRule(), emailRule())

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"min", "email"}, set.Names())
	})

	t.Run("rejects a duplicate name", func(t *testing.T) {
		// Arrange
		var set vogue.RuleSet
		require.NoError(t, set.Add(minRule()))

		// Act
		err := set.Add(minRule())

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), `duplicate rule "min"`)
		assert.Equal(t, 1, set.Len())
	})

	t.Run("rejects an invalid rule", func(t *testing.T) {
		// Arrange
		var set vogue.RuleSet
		broken := minRule()
		broken.Message = ""

		// Act
		err := set.Add(broken)

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "message must not be empty")
		assert.Zero(t, set.Len())
	})
}

func TestRuleSet_Get(t *testing.T) {
	// Arrange
	var set vogue.RuleSet
	require.NoError(t, set.Add(minRule()))

	t.Run("found", func(t *testing.T) {
		// Act
		got, ok := set.Get("min")

		// Assert
		require.True(t, ok)
		assert.Equal(t, "min", got.Name)
	})

	t.Run("missing", func(t *testing.T) {
		// Act
		_, ok := set.Get("max")

		// Assert
		assert.False(t, ok)
	})
}

func TestRuleSet_Suggest(t *testing.T) {
	// Arrange
	var set vogue.RuleSet
	for _, name := range []string{"min", "max", "len", "email", "oneof", "alphanum"} {
		rule := minRule()
		rule.Name = name
		require.NoError(t, set.Add(rule))
	}

	cases := []struct {
		name string
		in   string
		want string
		ok   bool
	}{
		{name: "exact match", in: "min", want: "min", ok: true},
		{name: "substitution", in: "mim", want: "min", ok: true},
		{name: "transposition", in: "emial", want: "email", ok: true},
		{name: "insertion", in: "lenn", want: "len", ok: true},
		{name: "deletion", in: "onef", want: "oneof", ok: true},
		{name: "too far", in: "completelydifferent", ok: false},
		{name: "empty", in: "", ok: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got, ok := set.Suggest(tc.in)

			// Assert
			assert.Equal(t, tc.ok, ok)
			if tc.ok {
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestRuleSet_Suggest_deterministicOnTies(t *testing.T) {
	// Arrange: "mix" is at distance 1 from both "min" and "max".
	var set vogue.RuleSet
	for _, name := range []string{"min", "max"} {
		rule := minRule()
		rule.Name = name
		require.NoError(t, set.Add(rule))
	}

	for range 20 {
		// Act
		got, ok := set.Suggest("mix")

		// Assert
		require.True(t, ok)
		assert.Equal(t, "max", got, "ties must resolve alphabetically")
	}
}

func TestRuleSet_Rules(t *testing.T) {
	// Arrange
	var set vogue.RuleSet
	require.NoError(t, set.Add(minRule(), emailRule()))

	// Act
	got := set.Rules()

	// Assert
	require.Len(t, got, 2)
	assert.Equal(t, "min", got[0].Name)
	assert.Equal(t, "email", got[1].Name)
}

func TestParamSpec_String(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		spec vogue.ParamSpec
		want string
	}{
		{name: "none", spec: vogue.ParamSpec{}, want: "none"},
		{name: "required int", spec: vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt}, want: "required int"},
		{name: "optional list", spec: vogue.ParamSpec{Presence: vogue.ParamOptional, Type: vogue.ParamList}, want: "optional list"},
		{name: "required regex", spec: vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamRegex}, want: "required regex"},
		{name: "required string", spec: vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamString}, want: "required string"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := tc.spec.String()

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRule_RenderMessage(t *testing.T) {
	// Arrange
	rule := minRule()

	// Act
	got, err := rule.RenderMessage(vogue.MessageData{Field: "title", Param: "3"})

	// Assert
	require.NoError(t, err)
	assert.Equal(t, "title must be at least 3 characters", got)
}

// trimRule is a representative normalizer: its Emit returns a statement that
// rewrites the working variable instead of a boolean expression.
func trimRule() vogue.Rule {
	return vogue.Rule{
		Name:      "trim",
		Kinds:     vogue.Kinds(vogue.String),
		Doc:       "Removes leading and trailing whitespace.",
		Message:   "{{.Field}} is normalised by trimming whitespace",
		Normalize: true,
		Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.TrimSpace(" + c.Var + ")" },
	}
}

func TestRule_Normalize(t *testing.T) {
	t.Run("accepts a normalizer backed by an emitter", func(t *testing.T) {
		// Arrange
		rule := trimRule()

		// Act
		err := rule.Validate()

		// Assert
		require.NoError(t, err)
		assert.True(t, rule.Normalize)
	})

	t.Run("rejects a normalizer backed by a call", func(t *testing.T) {
		// Arrange
		rule := emailRule()
		rule.Normalize = true

		// Act
		err := rule.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "normalizing rule")
	})
}

func TestRule_ImportsAndEmitKind(t *testing.T) {
	t.Run("a rule declares the imports its emitted code needs", func(t *testing.T) {
		// Arrange
		rule := trimRule()
		rule.Imports = []string{"strings"}

		// Act
		err := rule.Validate()

		// Assert
		require.NoError(t, err)
		assert.Equal(t, []string{"strings"}, rule.Imports)
	})

	t.Run("rejects an empty import path", func(t *testing.T) {
		// Arrange
		rule := trimRule()
		rule.Imports = []string{""}

		// Act
		err := rule.Validate()

		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "import path must not be empty")
	})

	t.Run("the emit context carries the kind being generated", func(t *testing.T) {
		// Arrange
		rule := vogue.Rule{
			Name:    "min",
			Kinds:   vogue.Kinds(vogue.String, vogue.Int),
			Message: "{{.Field}} is too small",
			Emit: func(c vogue.EmitContext) string {
				if c.Kind == vogue.Int {
					return c.Var + " >= " + c.Param
				}
				return "len(" + c.Var + ") >= " + c.Param
			},
			Param: vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
		}

		// Act
		got := rule.Emit(vogue.EmitContext{Var: "v", Param: "1", Field: "covers", Kind: vogue.Int})

		// Assert
		assert.Equal(t, "v >= 1", got)
	})
}

func TestExamples_TestGeneratorFields(t *testing.T) {
	t.Run("an example carries a note and the kinds it applies to", func(t *testing.T) {
		// Arrange
		example := vogue.Example{Kinds: vogue.Kinds(vogue.String), Param: "1", In: "a", Note: "a single rune"}

		// Act
		applies := example.AppliesTo(vogue.String, "1")

		// Assert
		assert.True(t, applies)
		assert.Equal(t, "a single rune", example.Note)
	})

	t.Run("an example without kinds applies to every kind", func(t *testing.T) {
		// Arrange
		example := vogue.Example{In: "a"}

		// Act & Assert
		assert.True(t, example.AppliesTo(vogue.String, ""))
		assert.True(t, example.AppliesTo(vogue.Int, ""))
	})

	t.Run("an example is scoped to the parameter it was written against", func(t *testing.T) {
		// Arrange
		example := vogue.Example{Param: "2", In: "ab"}

		// Act & Assert
		assert.False(t, example.AppliesTo(vogue.String, "3"))
		assert.True(t, example.AppliesTo(vogue.String, "2"))
	})

	t.Run("a rule declaring normalizations validates", func(t *testing.T) {
		// Arrange
		rule := vogue.Rule{
			Name:      "trim",
			Kinds:     vogue.Kinds(vogue.String),
			Message:   "{{.Field}} is trimmed",
			Normalize: true,
			Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.TrimSpace(" + c.Var + ")" },
			Examples: vogue.Examples{
				Normalized: []vogue.Normalization{{In: "  a  ", Out: "a", Note: "surrounding blanks"}},
			},
		}

		// Act
		err := rule.Validate()

		// Assert
		require.NoError(t, err)
		require.Len(t, rule.Examples.Normalized, 1)
		assert.Equal(t, "a", rule.Examples.Normalized[0].Out)
	})
}

func TestRule_Declare(t *testing.T) {
	t.Run("a rule may declare a package-level declaration alongside its expression", func(t *testing.T) {
		// Arrange
		rule := minRule()
		rule.Declare = func(c vogue.EmitContext) string {
			return "var _vogueBound = " + c.Param
		}

		// Act
		err := rule.Validate()

		// Assert
		require.NoError(t, err)
		require.NotNil(t, rule.Declare)
		assert.Equal(t, "var _vogueBound = 3", rule.Declare(vogue.EmitContext{Param: "3"}))
	})
}

func TestRule_Summary(t *testing.T) {
	t.Run("returns the first sentence of the documentation", func(t *testing.T) {
		// Arrange
		rule := vogue.Rule{Doc: "Rejects the empty string. It is checked where it is written."}

		// Act
		got := rule.Summary()

		// Assert
		assert.Equal(t, "Rejects the empty string.", got)
	})

	t.Run("returns a single-sentence documentation whole", func(t *testing.T) {
		// Arrange
		rule := vogue.Rule{Doc: "Rejects values below the bound."}

		// Act & Assert
		assert.Equal(t, "Rejects values below the bound.", rule.Summary())
	})

	t.Run("collapses the line breaks of a wrapped documentation", func(t *testing.T) {
		// Arrange
		rule := vogue.Rule{Doc: "Rejects the\n  empty string. And more."}

		// Act & Assert
		assert.Equal(t, "Rejects the empty string.", rule.Summary())
	})

	t.Run("reports an undocumented rule as empty", func(t *testing.T) {
		// Act & Assert
		assert.Empty(t, vogue.Rule{}.Summary())
	})
}
