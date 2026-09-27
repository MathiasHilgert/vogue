// Package rulecheck holds the runtime helpers the built-in vogue rules dispatch to.
//
// A rule whose check does not fit in a readable inline expression declares a
// [vogue.FuncRef] pointing here, and the generator emits a static call such as
// `rulecheck.Email(v)`. Everything in this package is therefore a pure predicate of
// the form `func(v string) bool`, reporting true when the value is acceptable:
// no state, no configuration and no error return, so a generated constructor
// stays a straight line of checks.
package rulecheck

import (
	"net/mail"
	"net/url"

	"github.com/google/uuid"
)

// Email reports whether v is a single, bare email address.
//
// The address is parsed with net/mail, which implements the RFC 5322 address
// grammar, and is then held to two further conditions that a stored address
// must satisfy: it must carry no display name, so `Waiter <a@b.test>` is
// rejected, and it must be exactly what was written, so surrounding whitespace
// or a redundant angle-bracket form is rejected rather than silently accepted
// in a shape the database would not round-trip. A list of two addresses is
// rejected too, because a field holds one address.
//
// Deliverability is not checked: no DNS lookup is performed, and a
// syntactically valid address at a domain that does not exist is accepted.
func Email(value string) bool {
	addr, err := mail.ParseAddress(value)
	return err == nil && addr.Name == "" && addr.Address == value
}

// URL reports whether v is an absolute http or https URL.
//
// The value is parsed with net/url as a request URI, so it must be absolute: a
// relative path such as `/menu` and a bare host such as `example.com` are both
// rejected, since neither can be resolved without knowing where it came from.
// The scheme must be http or https, compared case-insensitively as the parser
// normalises it, and the host must be present; other schemes such as `ftp:` or
// `javascript:` are rejected, which is what makes a stored URL safe to render
// as a link.
//
// Reachability is not checked: no request is made.
func URL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

// UUID reports whether v is a UUID in the canonical RFC 4122 text form,
// 8-4-4-4-12 lower- or upper-case hexadecimal digits separated by hyphens.
//
// Every version is accepted, including the nil UUID, so a value written by an
// older system can still be read. The alternative spellings google/uuid also
// parses — the undashed 32-digit form, the brace form and the `urn:uuid:`
// prefix — are deliberately rejected: a stored identifier that compares equal
// as text is worth more than one that has to be normalised before every
// comparison.
func UUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	return uuid.Validate(value) == nil
}
