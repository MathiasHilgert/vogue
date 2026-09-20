package parse

import (
	"strings"
	"unicode"
)

// FieldName derives the JSON and error field name of a value object from its Go
// type name. The result is the lower-camel spelling of the name, with
// initialisms folded so they read as one word: Title becomes "title", TabID
// becomes "tabId" and CUITNumber becomes "cuitNumber".
//
// The generator uses it for [FieldError.Field] and for the JSON representation,
// so a value object never has to restate its own name.
func FieldName(name string) string {
	words := splitWords(name)
	var b strings.Builder
	for i, w := range words {
		if i == 0 {
			b.WriteString(strings.ToLower(w))
			continue
		}
		b.WriteString(title(w))
	}
	return b.String()
}

// splitWords cuts a Go identifier into words at case boundaries, keeping a run
// of capitals that ends in a lowercase letter as two words: the initialism and
// the word it precedes.
func splitWords(name string) []string {
	runes := []rune(name)
	var (
		words []string
		start int
	)
	for i := 1; i < len(runes); i++ {
		if !unicode.IsUpper(runes[i]) {
			continue
		}
		prevIsUpper := unicode.IsUpper(runes[i-1])
		nextIsLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
		if prevIsUpper && !nextIsLower {
			continue
		}
		words = append(words, string(runes[start:i]))
		start = i
	}
	if start < len(runes) {
		words = append(words, string(runes[start:]))
	}
	return words
}

// title upper-cases the first rune of a word and lower-cases the rest, which is
// what turns the initialism "ID" into the camel-case segment "Id".
func title(w string) string {
	runes := []rune(strings.ToLower(w))
	if len(runes) == 0 {
		return ""
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

// constName builds the Go constant name of one enum value, as in
// TabStatus + "in_progress" giving TabStatusInProgress.
func constName(typeName, value string) string {
	var b strings.Builder
	b.WriteString(typeName)
	for _, part := range strings.Split(value, "_") {
		b.WriteString(title(part))
	}
	return b.String()
}
