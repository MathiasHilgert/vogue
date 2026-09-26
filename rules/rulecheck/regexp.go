package rulecheck

import (
	"regexp"
	"sync"
)

// patterns caches every pattern [Regexp] has compiled, keyed by its source.
// The set is closed: it holds the patterns written in the directives of the
// program, which is a handful, so it never needs evicting.
var patterns sync.Map

// Regexp reports whether v matches the RE2 pattern.
//
// The `regex` rule calls it rather than declaring a compiled pattern in the
// generated file, which keeps generated code free of package-level variables.
// The pattern is compiled on its first use and cached, so every later call
// only matches and allocates nothing. The generator has already compiled the
// pattern once to validate the directive, so a pattern reaching this function
// from generated code always compiles; one that does not is reported as a
// value that does not match rather than as a panic.
func Regexp(value, pattern string) bool {
	if cached, ok := patterns.Load(pattern); ok {
		compiled, _ := cached.(*regexp.Regexp)
		return compiled != nil && compiled.MatchString(value)
	}
	compiled, err := regexp.Compile(pattern)
	if err != nil {
		patterns.Store(pattern, (*regexp.Regexp)(nil))
		return false
	}
	patterns.Store(pattern, compiled)
	return compiled.MatchString(value)
}
