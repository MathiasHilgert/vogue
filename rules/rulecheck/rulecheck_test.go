package rulecheck_test

import (
	"testing"

	"github.com/MathiasHilgert/vogue/rules/rulecheck"
	"github.com/stretchr/testify/assert"
)

func TestEmail(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "an ordinary address", in: "waiter@example.com", want: true},
		{name: "a subaddressed mailbox", in: "orders+tab7@example.com", want: true},
		{name: "a subdomain", in: "a@mail.example.com", want: true},
		{name: "the empty string", in: "", want: false},
		{name: "a bare local part", in: "waiter", want: false},
		{name: "no domain", in: "waiter@", want: false},
		{name: "two at signs", in: "a@b@c.test", want: false},
		{name: "a display name, which a stored address must not carry", in: "Waiter <a@b.test>", want: false},
		{name: "surrounding whitespace", in: " a@b.test ", want: false},
		{name: "an address list", in: "a@b.test, c@d.test", want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Act
			got := rulecheck.Email(testCase.in)

			// Assert
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestURL(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "an https url", in: "https://example.com/menu", want: true},
		{name: "an http url with a port and a query", in: "http://example.com:8080/menu?tab=7", want: true},
		{name: "the empty string", in: "", want: false},
		{name: "a bare host without a scheme", in: "example.com", want: false},
		{name: "a relative path", in: "/menu", want: false},
		{name: "a scheme that is not http or https", in: "ftp://example.com", want: false},
		{name: "a scheme without a host", in: "https://", want: false},
		{name: "a control character", in: "https://example.com/\n", want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Act
			got := rulecheck.URL(testCase.in)

			// Assert
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestUUID(t *testing.T) {
	// Arrange
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{name: "a version 4 uuid", in: "9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a1b", want: true},
		{name: "a version 7 uuid", in: "018f3a2b-7c4d-7e8f-9a0b-1c2d3e4f5a6b", want: true},
		{name: "the nil uuid", in: "00000000-0000-0000-0000-000000000000", want: true},
		{name: "upper case hex", in: "9B2B4F52-1C2D-4E5A-9F3B-6D7C8E9F0A1B", want: true},
		{name: "the empty string", in: "", want: false},
		{name: "a truncated uuid", in: "9b2b4f52-1c2d-4e5a-9f3b", want: false},
		{name: "a non-hex rune", in: "9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a1z", want: false},
		{name: "the undashed form, which stored ids never use", in: "9b2b4f521c2d4e5a9f3b6d7c8e9f0a1b", want: false},
		{name: "surrounding whitespace", in: " 9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a1b ", want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// Act
			got := rulecheck.UUID(testCase.in)

			// Assert
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestRegexp(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		v       string
		pattern string
		want    bool
	}{
		{name: "matches an anchored pattern", v: "AR", pattern: "^[A-Z]{2}$", want: true},
		{name: "rejects a value outside the pattern", v: "ar", pattern: "^[A-Z]{2}$", want: false},
		{name: "matches unanchored anywhere", v: "x-42-y", pattern: "[0-9]+", want: true},
		{name: "rejects an invalid pattern rather than panicking", v: "a", pattern: "[", want: false},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			// Act
			got := rulecheck.Regexp(testCase.v, testCase.pattern)

			// Assert
			assert.Equal(t, testCase.want, got)
		})
	}
}

func TestRegexp_compilesOnce(t *testing.T) {
	// Arrange
	rulecheck.Regexp("warm", "^w")

	// Act
	allocs := testing.AllocsPerRun(100, func() { rulecheck.Regexp("warm", "^w") })

	// Assert
	assert.Zero(t, allocs, "a cached pattern must be matched without compiling it again")
}
