package vogue

import (
	"fmt"
	"strings"
	"sync"
	"text/template"
)

// MessageData is the data available to a rule message template. Templates use
// text/template syntax and may reference {{.Field}}, {{.Param}}, {{.Value}}
// and {{.Kind}}.
//
// The built-in messages do not name the field: a failure already carries it,
// so a message reads as what is wrong with it, "is required" or "must be a
// valid email address". Field stays available for a custom rule that wants it.
type MessageData struct {
	// Field is the value-object field name the rule was applied to.
	Field string
	// Kind is the kind of the value object the rule was applied to, so one
	// rule spanning several kinds can word its message for each, as `max`
	// does for a length and for a number. It is compared as text:
	// {{if eq .Kind.String "string"}}.
	Kind Kind
	// Param is the rule parameter as written in the directive.
	Param string
	// Value is the offending input rendered as text.
	Value string
}

// messageCacheEntry memoizes the outcome of compiling one template string,
// including a failure, so a broken template is reported without being parsed
// again on every call.
type messageCacheEntry struct {
	tmpl *template.Template
	err  error
}

// messageCache maps a template string to its compiled form. Rule messages come
// from a closed set of rule definitions, so the cache is bounded by the number
// of registered rules.
var messageCache sync.Map

// RenderMessage renders a rule message template with the given data. The
// template is compiled once per distinct template string and cached for the
// lifetime of the process, so rendering the same message repeatedly costs an
// execution and not a parse.
//
// text/template is used rather than html/template: messages are plain text and
// must not be HTML-escaped by the validation layer.
func RenderMessage(tmpl string, data MessageData) (string, error) {
	compiled, err := compileMessage(tmpl)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	if err := compiled.Execute(&b, data); err != nil {
		return "", fmt.Errorf("vogue: message template %q: %w", tmpl, err)
	}
	return b.String(), nil
}

// MustRenderMessage is like [RenderMessage] but panics when the template is
// invalid. It is meant for package-level rule definitions and tests, where a
// broken template is a programming error that should fail loudly at startup.
func MustRenderMessage(tmpl string, data MessageData) string {
	s, err := RenderMessage(tmpl, data)
	if err != nil {
		panic(err)
	}
	return s
}

// compileMessage returns the cached compilation of tmpl, compiling it on first
// use. Concurrent first uses may both compile; only one result is retained and
// both are equivalent.
func compileMessage(tmpl string) (*template.Template, error) {
	if cached, ok := messageCache.Load(tmpl); ok {
		entry := cached.(messageCacheEntry)
		return entry.tmpl, entry.err
	}

	entry := messageCacheEntry{}
	parsed, err := template.New("vogue.message").Option("missingkey=error").Parse(tmpl)
	if err != nil {
		entry.err = fmt.Errorf("vogue: message template %q: %w", tmpl, err)
	} else {
		entry.tmpl = parsed
	}

	messageCache.Store(tmpl, entry)
	return entry.tmpl, entry.err
}
