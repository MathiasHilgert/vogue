package vogue_test

import (
	"sync"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderMessage(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		tmpl string
		data vogue.MessageData
		want string
	}{
		{
			name: "all placeholders",
			tmpl: "{{.Field}} must be at least {{.Param}} characters, got {{.Value}}",
			data: vogue.MessageData{Field: "title", Param: "3", Value: "ab"},
			want: "title must be at least 3 characters, got ab",
		},
		{
			name: "no placeholder",
			tmpl: "must be a valid email address",
			data: vogue.MessageData{Field: "email"},
			want: "must be a valid email address",
		},
		{
			name: "missing data renders empty",
			tmpl: "{{.Field}}:{{.Param}}",
			data: vogue.MessageData{Field: "title"},
			want: "title:",
		},
		{
			name: "the kind the rule is used on",
			tmpl: `{{if eq .Kind.String "string"}}length {{end}}must be at most {{.Param}}`,
			data: vogue.MessageData{Kind: vogue.String, Param: "3"},
			want: "length must be at most 3",
		},
		{
			name: "another kind words it differently",
			tmpl: `{{if eq .Kind.String "string"}}length {{end}}must be at most {{.Param}}`,
			data: vogue.MessageData{Kind: vogue.Int, Param: "3"},
			want: "must be at most 3",
		},
		{
			name: "no html escaping",
			tmpl: "{{.Value}}",
			data: vogue.MessageData{Value: `a<b&c"`},
			want: `a<b&c"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got, err := vogue.RenderMessage(tc.tmpl, tc.data)

			// Assert
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestRenderMessage_errors(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		tmpl string
	}{
		{name: "unclosed action", tmpl: "{{.Field"},
		{name: "unknown field", tmpl: "{{.Nope}}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Act
			got, err := vogue.RenderMessage(tc.tmpl, vogue.MessageData{Field: "title"})

			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "vogue: message template")
			assert.Empty(t, got)
		})
	}
}

func TestRenderMessage_isCachedAndConcurrencySafe(t *testing.T) {
	// Arrange
	const tmpl = "{{.Field}} is invalid"
	var wg sync.WaitGroup

	// Act
	for range 32 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			got, err := vogue.RenderMessage(tmpl, vogue.MessageData{Field: "title"})
			assert.NoError(t, err)
			assert.Equal(t, "title is invalid", got)
		}()
	}
	wg.Wait()

	// Assert
	allocs := testing.AllocsPerRun(50, func() {
		_, _ = vogue.RenderMessage(tmpl, vogue.MessageData{Field: "title"})
	})
	assert.Less(t, allocs, 20.0, "a cached template must not recompile on every render")
}

func TestMustRenderMessage(t *testing.T) {
	t.Run("renders", func(t *testing.T) {
		// Act
		got := vogue.MustRenderMessage("{{.Field}} is required", vogue.MessageData{Field: "title"})

		// Assert
		assert.Equal(t, "title is required", got)
	})

	t.Run("panics on a broken template", func(t *testing.T) {
		// Act & Assert
		assert.Panics(t, func() {
			vogue.MustRenderMessage("{{.Field", vogue.MessageData{})
		})
	})
}
