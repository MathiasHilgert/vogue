package parse

import (
	"strings"
	"unicode"
)

// FieldName derives the JSON and error field name of a value object from its Go
// type name. The result is the lower-camel spelling Go itself uses: the first
// word is lower-cased whole, and every later word keeps its case, so an
// initialism stays an initialism. Title becomes "title", TabID becomes
// "tabID", SubdivisionISOCode becomes "subdivisionISOCode" and CUITNumber
// becomes "cuitNumber", the way golint spells the same names.
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
		b.WriteString(w)
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
// what turns the lower-snake enum value "in" into the method segment "In".
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
	return typeName + methodName(value)
}

// methodName returns the catalogue method of an enum member: its lower-snake
// value in PascalCase, so "in_progress" becomes InProgress.
func methodName(value string) string {
	var b strings.Builder
	for _, part := range strings.Split(value, "_") {
		b.WriteString(title(part))
	}
	return b.String()
}

// plural returns the English plural of a Go type name, which is what the
// catalogue type of an enum is called. It covers the regular forms — a
// sibilant takes -es, a consonant followed by y takes -ies, everything else
// takes -s — and nothing more: an enum whose name has an irregular plural
// still gets a regular, predictable catalogue name.
func plural(name string) string {
	lower := strings.ToLower(name)
	for _, sibilant := range []string{"s", "x", "z", "ch", "sh"} {
		if strings.HasSuffix(lower, sibilant) {
			return name + "es"
		}
	}
	if n := len(lower); n > 1 && lower[n-1] == 'y' && !strings.ContainsRune("aeiou", rune(lower[n-2])) {
		return name[:len(name)-1] + "ies"
	}
	return name + "s"
}
