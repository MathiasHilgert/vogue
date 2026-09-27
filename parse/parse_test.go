package parse_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const goodSrc = `package tab

// Title is the human-readable name of a tab.
// It is bounded on both ends.
//vogue:string  Title      required trim min=1 max=120

//vogue:string Email required lower email
//vogue:int    Covers min=1 max=200
//vogue:enum   TabStatus open,in_progress,closed
//vogue:id     TabID
`

func TestFiles(t *testing.T) {
	t.Parallel()

	// Arrange / Act.
	pkg, err := parseSrc(t, goodSrc)

	// Assert.
	require.NoError(t, err)
	assert.Equal(t, "tab", pkg.Name)
	require.Len(t, pkg.Files, 1)
	assert.Equal(t, "vo.go", pkg.Files[0].Path)

	directives := pkg.Files[0].Directives
	require.Len(t, directives, 5)

	t.Run("string directive", func(t *testing.T) {
		t.Parallel()

		d := directives[0]

		assert.Equal(t, vogue.String, d.Kind)
		assert.Equal(t, "Title", d.Name)
		assert.Equal(t, "title", d.Field)
		assert.Equal(t, "Title is the human-readable name of a tab.\nIt is bounded on both ends.", d.Doc)
		assert.Equal(t, []string{"required", "trim", "min=1", "max=120"}, ruleUses(d))
		assert.Equal(t, 5, d.Pos.Line)
		assert.Equal(t, 1, d.Pos.Column)
	})

	t.Run("rule positions point at the token", func(t *testing.T) {
		t.Parallel()

		d := directives[0]

		require.Len(t, d.Rules, 4)
		assert.Equal(t, at(t, goodSrc, "required trim"), d.Rules[0].Pos.String())
		assert.Equal(t, at(t, goodSrc, "min=1 max=120"), d.Rules[2].Pos.String())
	})

	t.Run("doc is empty without preceding comment lines", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, directives[1].Doc)
		assert.Equal(t, "email", directives[1].Field)
	})

	t.Run("int directive", func(t *testing.T) {
		t.Parallel()

		d := directives[2]

		assert.Equal(t, vogue.Int, d.Kind)
		assert.Equal(t, "covers", d.Field)
		assert.Equal(t, []string{"min=1", "max=200"}, ruleUses(d))
	})

	t.Run("enum directive derives constants", func(t *testing.T) {
		t.Parallel()

		d := directives[3]

		assert.Equal(t, vogue.Enum, d.Kind)
		assert.Empty(t, d.Rules)
		assert.Equal(t, []parse.EnumValue{
			{Value: "open", Const: "TabStatusOpen", Method: "Open"},
			{Value: "in_progress", Const: "TabStatusInProgress", Method: "InProgress"},
			{Value: "closed", Const: "TabStatusClosed", Method: "Closed"},
		}, d.Values)
		assert.Equal(t, "TabStatuses", d.Catalogue)
	})

	t.Run("id directive", func(t *testing.T) {
		t.Parallel()

		d := directives[4]

		assert.Equal(t, vogue.ID, d.Kind)
		assert.Equal(t, "TabID", d.Name)
		assert.Equal(t, "tabId", d.Field)
		assert.Equal(t, parse.IDUUIDv7, d.Strategy)
	})
}

func TestFilesIgnoresNonDirectives(t *testing.T) {
	t.Parallel()

	// Arrange.
	src := "package tab\n\n// ordinary comment\n/* //vogue:string Block */\nvar _ = 1 // trailing\n"

	// Act.
	pkg, err := parseSrc(t, src)

	// Assert.
	require.NoError(t, err)
	require.Len(t, pkg.Files, 1)
	assert.Empty(t, pkg.Files[0].Directives)
}

func TestFilesErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		src    string
		needle string
		want   string
	}{
		{
			name:   "unknown rule suggests the nearest",
			src:    "package tab\n//vogue:string Title required mni=1\n",
			needle: "mni=1",
			want:   `unknown rule "mni" for kind string (did you mean "min"?)`,
		},
		{
			name:   "unknown rule without a near match",
			src:    "package tab\n//vogue:string Title zzzzzz\n",
			needle: "zzzzzz",
			want:   `unknown rule "zzzzzz" for kind string`,
		},
		{
			name:   "rule does not apply to the kind",
			src:    "package tab\n//vogue:int Covers email\n",
			needle: "email",
			want:   `rule "email" does not apply to kind int (it applies to: string)`,
		},
		{
			name:   "missing required parameter",
			src:    "package tab\n//vogue:string Title min\n",
			needle: "min",
			want:   `rule "min" requires a parameter (write min=<int>)`,
		},
		{
			name:   "unexpected parameter",
			src:    "package tab\n//vogue:string Title trim=1\n",
			needle: "trim=1",
			want:   `rule "trim" takes no parameter (write trim)`,
		},
		{
			name:   "parameter is not an integer",
			src:    "package tab\n//vogue:string Title min=abc\n",
			needle: "min=abc",
			want:   `invalid parameter for rule "min": "abc" is not an integer`,
		},
		{
			name:   "empty list item",
			src:    "package tab\n//vogue:string Title oneof=a,,b\n",
			needle: "oneof=a,,b",
			want:   `invalid parameter for rule "oneof": item 2 of "a,,b" is empty`,
		},
		{
			name:   "invalid type name",
			src:    "package tab\n//vogue:string title\n",
			needle: "title",
			want:   `invalid type name "title" (want ^[A-Z][A-Za-z0-9]*$)`,
		},
		{
			name:   "missing type name",
			src:    "package tab\n//vogue:string\n",
			needle: "//vogue:string",
			want:   `missing type name after kind "string"`,
		},
		{
			name:   "unknown kind",
			src:    "package tab\n//vogue:strng Title\n",
			needle: "//vogue:strng",
			want:   `unknown kind "strng" (did you mean "string"?)`,
		},
		{
			name:   "enum takes one list",
			src:    "package tab\n//vogue:enum TabStatus open closed\n",
			needle: "closed",
			want:   `enum "TabStatus" takes exactly one comma-separated list of values`,
		},
		{
			name:   "enum needs values",
			src:    "package tab\n//vogue:enum TabStatus\n",
			needle: "TabStatus",
			want:   `enum "TabStatus" requires a comma-separated list of values`,
		},
		{
			name:   "enum needs at least two values",
			src:    "package tab\n//vogue:enum TabStatus open\n",
			needle: "open",
			want:   `enum "TabStatus" must declare at least 2 values, got 1`,
		},
		{
			name:   "invalid enum value",
			src:    "package tab\n//vogue:enum TabStatus Open,closed\n",
			needle: "Open,closed",
			want:   `invalid enum value "Open" (want ^[a-z][a-z0-9_]*$)`,
		},
		{
			name:   "duplicate enum value",
			src:    "package tab\n//vogue:enum TabStatus open,open\n",
			needle: "open,open",
			want:   `duplicate enum value "open"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Arrange / Act.
			pkg, err := parseSrc(t, tt.src)

			// Assert.
			require.Error(t, err)
			assert.Nil(t, pkg)
			assert.Equal(t, at(t, tt.src, tt.needle)+": "+tt.want, err.Error())
		})
	}
}

func TestFilesDuplicateRule(t *testing.T) {
	t.Parallel()

	// Arrange.
	src := "package tab\n//vogue:string Title min=1 max=2 min=3\n"

	// Act.
	_, err := parseSrc(t, src)

	// Assert.
	require.Error(t, err)
	assert.Equal(t,
		at(t, src, "min=3")+`: duplicate rule "min" (already given at `+at(t, src, "min=1")+")",
		err.Error())
}

func TestFilesDuplicateName(t *testing.T) {
	t.Parallel()

	// Arrange.
	src := "package tab\n//vogue:string Title min=1\n//vogue:int Title\n"

	// Act.
	_, err := parseSrc(t, src)

	// Assert.
	require.Error(t, err)
	assert.Equal(t,
		at(t, src, "//vogue:int Title")+`: duplicate value object "Title" (first declared at `+at(t, src, "//vogue:string Title")+")",
		err.Error())
}

func TestFilesEnumCatalogue(t *testing.T) {
	t.Parallel()

	t.Run("names the catalogue after the plural of the enum", func(t *testing.T) {
		t.Parallel()

		cases := []struct{ name, want string }{
			{name: "PlaceKind", want: "PlaceKinds"},
			{name: "TabStatus", want: "TabStatuses"},
			{name: "Category", want: "Categories"},
			{name: "Day", want: "Days"},
			{name: "Match", want: "Matches"},
			{name: "Box", want: "Boxes"},
			{name: "Wish", want: "Wishes"},
		}
		for _, tc := range cases {
			// Act.
			pkg, err := parseSrc(t, "package tab\n//vogue:enum "+tc.name+" a,b\n")

			// Assert.
			require.NoError(t, err)
			assert.Equal(t, tc.want, pkg.Files[0].Directives[0].Catalogue, tc.name)
		}
	})

	t.Run("rejects a member whose method collides with a catalogue method", func(t *testing.T) {
		t.Parallel()

		// Act.
		_, err := parseSrc(t, "package tab\n//vogue:enum Scope all,mine\n")

		// Assert.
		require.Error(t, err)
		assert.Contains(t, err.Error(), `enum member "all" would be generated as the method All`)
	})

	t.Run("rejects a catalogue that collides with another value object", func(t *testing.T) {
		t.Parallel()

		// Arrange.
		src := "package tab\n//vogue:enum Status a,b\n//vogue:string Statuses required\n"

		// Act.
		_, err := parseSrc(t, src)

		// Assert.
		require.Error(t, err)
		assert.Contains(t, err.Error(), `the catalogue type Statuses of enum "Status" collides with a value object`)
	})
}

func TestFilesExamples(t *testing.T) {
	t.Parallel()

	t.Run("collects example tokens apart from the rules", func(t *testing.T) {
		t.Parallel()

		// Act.
		pkg, err := parseSrc(t, "package tab\n//vogue:string Code trim required example=AR example=\"DE\"\n")

		// Assert.
		require.NoError(t, err)
		d := pkg.Files[0].Directives[0]
		assert.Equal(t, []string{"trim", "required"}, ruleUses(d))
		assert.Equal(t, []string{"AR", "DE"}, d.Examples)
	})

	t.Run("rejects an example the kind cannot express", func(t *testing.T) {
		t.Parallel()

		// Act.
		_, err := parseSrc(t, "package tab\n//vogue:int Covers min=1 example=many\n")

		// Assert.
		require.Error(t, err)
		assert.Contains(t, err.Error(), `example "many" is not a valid int`)
	})

	t.Run("rejects an example without a value", func(t *testing.T) {
		t.Parallel()

		// Act.
		_, err := parseSrc(t, "package tab\n//vogue:string Code required example\n")

		// Assert.
		require.Error(t, err)
		assert.Contains(t, err.Error(), "example requires a value")
	})
}

func TestFilesInvalidRegexParam(t *testing.T) {
	t.Parallel()

	// Arrange.
	src := "package tab\n//vogue:string Title regex=\"^[a-z\"\n"

	// Act.
	_, err := parseSrc(t, src)

	// Assert.
	require.Error(t, err)
	assert.Equal(t,
		at(t, src, `regex="^[a-z"`)+`: invalid parameter for rule "regex": `+
			"error parsing regexp: missing closing ]: `[a-z`",
		err.Error())
}

func TestFilesReportsEveryProblem(t *testing.T) {
	t.Parallel()

	// Arrange.
	src := "package tab\n//vogue:string Title mni=1\n//vogue:int Covers email\n"

	// Act.
	_, err := parseSrc(t, src)

	// Assert.
	require.Error(t, err)
	var errs parse.Errors
	require.ErrorAs(t, err, &errs)
	require.Len(t, errs, 2)
	assert.Equal(t, 2, errs[0].Pos.Line)
	assert.Equal(t, 3, errs[1].Pos.Line)
}

func TestFilesRejectsUnterminatedQuote(t *testing.T) {
	t.Parallel()

	// Arrange.
	src := "package tab\n//vogue:string Title regex=\"^a\n"

	// Act.
	_, err := parseSrc(t, src)

	// Assert.
	require.Error(t, err)
	assert.Equal(t, at(t, src, "//vogue:string")+": unterminated quoted parameter", err.Error())
}
