package rules_test

import (
	"sort"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	t.Run("every shipped rule is internally consistent", func(t *testing.T) {
		// Arrange
		all := rules.All()
		require.NotEmpty(t, all)

		for _, rule := range all {
			t.Run(rule.Name, func(t *testing.T) {
				// Act
				err := rule.Validate()

				// Assert
				assert.NoError(t, err)
			})
		}
	})

	t.Run("ships the documented catalogue", func(t *testing.T) {
		// Arrange
		want := []string{
			"alpha", "alphanum", "ascii", "contains", "email", "excludes", "len",
			"lower", "max", "min", "multipleof", "nonneg", "nonzero", "nospace",
			"numeric", "oneof", "positive", "prefix", "printable", "regex",
			"required", "scale", "squish", "suffix", "trim", "upper", "url", "uuid",
		}

		// Act
		got := make([]string, 0, len(rules.All()))
		for _, rule := range rules.All() {
			got = append(got, rule.Name)
		}
		sort.Strings(got)

		// Assert
		assert.Equal(t, want, got)
	})

	t.Run("every rule is documented and carries examples a test can be built from", func(t *testing.T) {
		// Arrange
		for _, rule := range rules.All() {
			t.Run(rule.Name, func(t *testing.T) {
				// Assert
				assert.NotEmpty(t, rule.Doc, "a rule without documentation cannot be catalogued")
				assert.NotEmpty(t, rule.Message, "a rule without a message cannot report a failure")

				if rule.Normalize {
					assert.GreaterOrEqual(t, len(rule.Examples.Normalized), 2,
						"a normalizer must show at least two rewrites")
					return
				}
				assert.GreaterOrEqual(t, len(rule.Examples.Valid), 2,
					"a check must show at least two accepted inputs")

				// `required` is the one rule whose rejected inputs are
				// exhaustible: the empty string is the only value it refuses,
				// and inventing a second example would only pad the table.
				wantInvalid := 2
				if rule.Name == "required" {
					wantInvalid = 1
				}
				assert.GreaterOrEqual(t, len(rule.Examples.Invalid), wantInvalid,
					"a check must show every rejected input it can")
			})
		}
	})

	t.Run("every example names the kinds and the parameter it belongs to", func(t *testing.T) {
		// Arrange
		for _, rule := range rules.All() {
			t.Run(rule.Name, func(t *testing.T) {
				all := append(append([]vogue.Example{}, rule.Examples.Valid...), rule.Examples.Invalid...)
				for _, example := range all {
					// Assert
					assert.NotEmpty(t, example.Note, "an example without a note reads badly as a subtest name")
					if rule.Param.Presence == vogue.ParamNone {
						assert.Empty(t, example.Param, "a rule without a parameter has no parameter-bound example")
					} else {
						assert.NotEmpty(t, example.Param, "a parameterised rule needs parameter-bound examples")
					}
					if len(rule.Kinds.Kinds()) > 1 {
						assert.False(t, example.Kinds.Empty(),
							"a rule spanning several kinds needs kind-scoped examples")
					}
					for _, kind := range example.Kinds.Kinds() {
						assert.True(t, rule.Kinds.Has(kind),
							"example declares kind %s the rule does not apply to", kind)
					}
				}
			})
		}
	})

	t.Run("returns a fresh slice, so a caller cannot corrupt the catalogue", func(t *testing.T) {
		// Arrange
		first := rules.All()

		// Act
		first[0] = vogue.Rule{}

		// Assert
		assert.NotEqual(t, vogue.Rule{}, rules.All()[0])
	})
}

func TestSet(t *testing.T) {
	t.Run("builds a set holding every shipped rule", func(t *testing.T) {
		// Act
		set, err := rules.Set()

		// Assert
		require.NoError(t, err)
		require.NotNil(t, set)
		assert.Equal(t, len(rules.All()), set.Len())

		for _, rule := range rules.All() {
			got, ok := set.Get(rule.Name)
			require.True(t, ok, "rule %q is missing from the set", rule.Name)
			assert.Equal(t, rule.Name, got.Name)
		}
	})

	t.Run("suggests the nearest name for a typo", func(t *testing.T) {
		// Arrange
		set := rules.MustSet()

		// Act
		got, ok := set.Suggest("emial")

		// Assert
		require.True(t, ok)
		assert.Equal(t, "email", got)
	})

	t.Run("returns an independent set on every call", func(t *testing.T) {
		// Act
		first, second := rules.MustSet(), rules.MustSet()

		// Assert
		assert.NotSame(t, first, second)
	})
}

func TestMustSet(t *testing.T) {
	t.Run("returns the catalogue without an error to handle", func(t *testing.T) {
		// Act
		set := rules.MustSet()

		// Assert
		require.NotNil(t, set)
		assert.Equal(t, len(rules.All()), set.Len())
	})
}

func TestMessages(t *testing.T) {
	// Arrange
	cases := []struct {
		name  string
		rule  string
		param string
		want  string
	}{
		{name: "required", rule: "required", want: "title is required"},
		{name: "min on a string", rule: "min", param: "3", want: "title must be at least 3"},
		{name: "max on a string", rule: "max", param: "120", want: "title must be at most 120"},
		{name: "len", rule: "len", param: "3", want: "title must be exactly 3 characters long"},
		{name: "email", rule: "email", want: "title must be a valid email address"},
		{name: "oneof", rule: "oneof", param: "a,b", want: "title must be one of: a,b"},
		{name: "prefix", rule: "prefix", param: "SKU-", want: `title must start with "SKU-"`},
	}

	set := rules.MustSet()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange
			rule, ok := set.Get(tc.rule)
			require.True(t, ok)

			// Act
			got, err := rule.RenderMessage(vogue.MessageData{Field: "title", Param: tc.param})

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestMessages_NeverReferenceTheRuntimeValue(t *testing.T) {
	// Arrange
	for _, rule := range rules.All() {
		t.Run(rule.Name, func(t *testing.T) {
			// Act
			got, err := rule.RenderMessage(vogue.MessageData{Field: "f", Param: "1", Value: "SECRET"})

			// Assert
			require.NoError(t, err)
			assert.NotContains(t, got, "SECRET",
				"messages are rendered at generate time, so they cannot carry the runtime value")
		})
	}
}

func TestEmit(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		rule vogue.Rule
		ctx  vogue.EmitContext
		want string
	}{
		{
			name: "min counts runes on a string",
			rule: rules.Min,
			ctx:  vogue.EmitContext{Var: "v", Param: "3", Kind: vogue.String, Ident: "minParam"},
			want: "utf8.RuneCountInString(v) >= minParam",
		},
		{
			name: "min compares the value itself on an integer",
			rule: rules.Min,
			ctx:  vogue.EmitContext{Var: "v", Param: "3", Kind: vogue.Int, Ident: "minParam"},
			want: "v >= minParam",
		},
		{
			name: "oneof quotes string items",
			rule: rules.OneOf,
			ctx:  vogue.EmitContext{Var: "v", Param: "eur,usd", Kind: vogue.String},
			want: `v == "eur" || v == "usd"`,
		},
		{
			name: "oneof looks an integer up in a literal slice",
			rule: rules.OneOf,
			ctx:  vogue.EmitContext{Var: "v", Param: "1,2", Kind: vogue.Int},
			want: "slices.Contains([]int64{1, 2}, v)",
		},
		{
			name: "multipleof divides",
			rule: rules.MultipleOf,
			ctx:  vogue.EmitContext{Var: "v", Param: "15", Kind: vogue.Int, Ident: "multipleofParam"},
			want: "v%multipleofParam == 0",
		},
		{
			name: "multipleof of zero accepts only zero, since a modulo by zero would not compile",
			rule: rules.MultipleOf,
			ctx:  vogue.EmitContext{Var: "v", Param: "0", Kind: vogue.Int},
			want: "v == 0",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := tc.rule.Emit(tc.ctx)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestLocal(t *testing.T) {
	cases := []struct {
		name string
		rule vogue.Rule
		ctx  vogue.EmitContext
		want string
	}{
		{
			name: "a rune bound is a named constant",
			rule: rules.Max,
			ctx:  vogue.EmitContext{Param: "120", Kind: vogue.String, Ident: "maxParam"},
			want: "const maxParam = 120",
		},
		{
			name: "a decimal bound is its coefficient and its scale",
			rule: rules.Min,
			ctx:  vogue.EmitContext{Param: "-90.5", Kind: vogue.Decimal, Ident: "minParam"},
			want: "const (\n\tminParamCoef  = -905\n\tminParamScale = 1\n)",
		},
		{
			name: "a scale is a named constant",
			rule: rules.Scale,
			ctx:  vogue.EmitContext{Param: "6", Kind: vogue.Decimal, Ident: "scaleParam"},
			want: "const scaleParam = 6",
		},
		{
			name: "multipleof of zero declares nothing",
			rule: rules.MultipleOf,
			ctx:  vogue.EmitContext{Param: "0", Kind: vogue.Int, Ident: "multipleofParam"},
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got := tc.rule.Local(tc.ctx)

			// Assert
			assert.Equal(t, tc.want, got)
		})
	}

	t.Run("a decimal bound is compared against the coefficient it declares", func(t *testing.T) {
		// Act
		got := rules.Min.Emit(vogue.EmitContext{Var: "v", Param: "-90.5", Kind: vogue.Decimal, Ident: "minParam"})

		// Assert
		assert.Equal(t, "v.Cmp(decimal.MustNew(minParamCoef, minParamScale)) >= 0", got)
	})
}

func TestRegex(t *testing.T) {
	t.Run("calls the caching matcher instead of declaring a package-level pattern", func(t *testing.T) {
		// Assert
		assert.Nil(t, rules.Regex.Declare)
		require.NotNil(t, rules.Regex.Call)
		assert.Equal(t, "fn.Regexp", rules.Regex.Call.Selector())
	})
}
