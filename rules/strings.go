package rules

import (
	"strconv"

	"github.com/MathiasHilgert/vogue"
)

// str is the set of kinds a string-only rule applies to, named once so the
// catalogue reads as a list of rules rather than a list of set literals.
var str = vogue.Kinds(vogue.String)

// Required rejects the empty value.
var Required = vogue.Rule{
	Name:  "required",
	Kinds: str,
	Doc: "Rejects the empty string. It is checked at the point it is written, so a `required` " +
		"after `trim` rejects a value that was nothing but whitespace, while a `required` before " +
		"it would have accepted it. Nothing else in the catalogue rejects an empty value, which " +
		"is what lets an optional field still be checked for shape when it is filled in.",
	Message: "{{.Field}} is required",
	Emit:    func(c vogue.EmitContext) string { return c.Var + ` != ""` },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "Tortilla", Note: "an ordinary value"},
			{In: " ", Note: "a blank, which only a preceding trim turns into a failure"},
		},
		Invalid: []vogue.Example{
			{In: "", Note: "the empty string"},
		},
	},
}

// Min bounds a value from below: a rune count for a string, the magnitude for
// an integer.
var Min = vogue.Rule{
	Name:  "min",
	Kinds: vogue.Kinds(vogue.String, vogue.Int),
	Doc: "Rejects values below the bound. On a string the bound is a count of runes, not of " +
		"bytes, so `min=3` accepts \"añó\": it counts what a person filling in the form counts. " +
		"On an integer it is the value itself, and the bound is inclusive on both kinds. Place it " +
		"after any normalizer, so the bound is measured on the value that will actually be stored.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
	Message: "{{.Field}} must be at least {{.Param}}",
	Imports: []string{importUTF8},
	Emit:    func(c vogue.EmitContext) string { return compare(c, ">=") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Kinds: str, Param: "1", In: "a", Note: "a single rune meets a bound of one"},
			{Kinds: str, Param: "3", In: "añó", Note: "three accented runes, six bytes, count as three"},
			{Kinds: intK, Param: "1", In: "1", Note: "the bound itself is accepted"},
			{Kinds: intK, Param: "1", In: "42", Note: "a table well above the bound"},
		},
		Invalid: []vogue.Example{
			{Kinds: str, Param: "1", In: "", Note: "the empty string is shorter than one rune"},
			{Kinds: str, Param: "3", In: "ab", Note: "two runes fall short of three"},
			{Kinds: intK, Param: "1", In: "0", Note: "a table with nobody at it"},
			{Kinds: intK, Param: "1", In: "-5", Note: "a negative count is below any positive bound"},
		},
	},
}

// Max bounds a value from above and is the counterpart of [Min].
var Max = vogue.Rule{
	Name:  "max",
	Kinds: vogue.Kinds(vogue.String, vogue.Int),
	Doc: "Rejects values above the bound, counting runes on a string and the value itself on an " +
		"integer, inclusive on both. Because it counts runes, a column declared varchar(120) is " +
		"better guarded by a `max` on the byte length of its widest expected encoding; `max=120` " +
		"here means 120 characters as a person counts them, which is what a form should say.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
	Message: "{{.Field}} must be at most {{.Param}}",
	Imports: []string{importUTF8},
	Emit:    func(c vogue.EmitContext) string { return compare(c, "<=") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Kinds: str, Param: "4", In: "abcd", Note: "the bound itself is accepted"},
			{Kinds: str, Param: "4", In: "", Note: "the empty string is under every bound"},
			{Kinds: intK, Param: "200", In: "200", Note: "a full house is still a house"},
			{Kinds: intK, Param: "200", In: "-1", Note: "a negative value is under every positive bound"},
		},
		Invalid: []vogue.Example{
			{Kinds: str, Param: "4", In: "abcde", Note: "one rune more than the bound holds"},
			{Kinds: str, Param: "4", In: "añóra", Note: "five accented runes are five, not ten"},
			{Kinds: intK, Param: "200", In: "201", Note: "one guest more than the room holds"},
			{Kinds: intK, Param: "200", In: "1000", Note: "a value far above the bound"},
		},
	},
}

// Len fixes the exact rune count of a value.
var Len = vogue.Rule{
	Name:  "len",
	Kinds: str,
	Doc: "Requires the value to be exactly the given number of runes long. It is the rule for a " +
		"code of fixed width — a three-letter currency code, a two-letter country code — where a " +
		"value of any other length is a typo rather than a shorter name. Runes are counted, not " +
		"bytes, so `len=3` accepts \"añó\" and rejects a value that merely encodes to three bytes.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
	Message: "{{.Field}} must be exactly {{.Param}} characters long",
	Imports: []string{importUTF8},
	Emit:    func(c vogue.EmitContext) string { return compare(c, "==") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: "3", In: "EUR", Note: "a currency code of the expected width"},
			{Param: "3", In: "añó", Note: "three runes, six bytes, still three characters"},
		},
		Invalid: []vogue.Example{
			{Param: "3", In: "EU", Note: "a code one character short"},
			{Param: "3", In: "EURO", Note: "a code one character long"},
			{Param: "3", In: "", Note: "the empty string, which has no characters at all"},
		},
	},
}

// Email requires a single, bare email address.
var Email = vogue.Rule{
	Name:  "email",
	Kinds: str,
	Doc: "Requires a single email address in the RFC 5322 grammar, parsed by net/mail rather " +
		"than matched against a regular expression. A display name is rejected, so " +
		"\"Waiter <a@b.test>\" does not pass: what is stored must be the address itself. " +
		"Surrounding whitespace and a list of two addresses are rejected for the same reason. " +
		"Deliverability is not checked — no DNS lookup happens — so a well-formed address at a " +
		"domain that does not exist is accepted. Pair it with `lower` to store one canonical " +
		"spelling.",
	Message: "{{.Field}} must be a valid email address",
	Call:    &vogue.FuncRef{Path: importFn, Name: "Email"},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "waiter@example.com", Note: "an ordinary address"},
			{In: "orders+tab7@example.com", Note: "a subaddressed mailbox"},
		},
		Invalid: []vogue.Example{
			{In: "waiter", Note: "a local part with no domain"},
			{In: "Waiter <a@b.test>", Note: "a display name, which a stored address must not carry"},
			{In: "", Note: "the empty string"},
		},
	},
}

// URL requires an absolute http or https URL.
var URL = vogue.Rule{
	Name:  "url",
	Kinds: str,
	Doc: "Requires an absolute http or https URL, parsed by net/url as a request URI. A relative " +
		"path such as \"/menu\" and a bare host such as \"example.com\" are rejected, because " +
		"neither can be resolved without knowing where it came from, and a scheme other than http " +
		"or https is rejected too, which is what makes a stored URL safe to render as a link. " +
		"Reachability is not checked: no request is made.",
	Message: "{{.Field}} must be a valid http or https URL",
	Call:    &vogue.FuncRef{Path: importFn, Name: "URL"},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "https://example.com/menu", Note: "an https address"},
			{In: "http://example.com:8080/menu?tab=7", Note: "a port and a query string are fine"},
		},
		Invalid: []vogue.Example{
			{In: "example.com", Note: "a bare host with no scheme"},
			{In: "ftp://example.com", Note: "a scheme that is not http or https"},
			{In: "", Note: "the empty string"},
		},
	},
}

// UUID requires a UUID in canonical text form.
var UUID = vogue.Rule{
	Name:  "uuid",
	Kinds: str,
	Doc: "Requires a UUID written in the canonical RFC 4122 form, 8-4-4-4-12 hexadecimal digits " +
		"separated by hyphens, in either case. Every version is accepted, including the nil UUID, " +
		"so an identifier written by an older system can still be read. The alternative spellings " +
		"— the undashed 32-digit form, the brace form and the `urn:uuid:` prefix — are rejected on " +
		"purpose: an identifier that compares equal as text is worth more than one that has to be " +
		"normalised before every comparison. For an identifier of your own use the `id` kind " +
		"instead, which gives it its own type.",
	Message: "{{.Field}} must be a valid UUID",
	Call:    &vogue.FuncRef{Path: importFn, Name: "UUID"},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "9b2b4f52-1c2d-4e5a-9f3b-6d7c8e9f0a1b", Note: "a version 4 identifier"},
			{In: "018f3a2b-7c4d-7e8f-9a0b-1c2d3e4f5a6b", Note: "a time-ordered version 7 identifier"},
		},
		Invalid: []vogue.Example{
			{In: "9b2b4f521c2d4e5a9f3b6d7c8e9f0a1b", Note: "the undashed form, which stored ids never use"},
			{In: "not-a-uuid", Note: "a value that is not hexadecimal at all"},
			{In: "", Note: "the empty string"},
		},
	},
}

// Regex requires the value to match a pattern compiled once, at start-up.
var Regex = vogue.Rule{
	Name:  "regex",
	Kinds: str,
	Doc: "Requires the value to match the given RE2 pattern. The pattern is compiled at generate " +
		"time, so a malformed one is a generator error naming the directive rather than a panic in " +
		"production, and it is compiled again into a package-level variable in the generated file, " +
		"so the constructor only matches. The match is unanchored: write ^ and $ when the whole " +
		"value must match. The pattern may not contain a space, because a directive is read as " +
		"whitespace-separated tokens; use `[[:space:]]` or `\\s` for one. Reach for a named rule " +
		"first — `alphanum`, `prefix`, `len` — and keep this for a shape that has no name.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamRegex},
	Message: "{{.Field}} must match the pattern {{.Param}}",
	Imports: []string{importRegexp},
	Declare: func(c vogue.EmitContext) string {
		return "// " + regexpVar(c.Param) + " is the pattern the `regex` rule matches against,\n" +
			"// compiled once at start-up rather than on every call.\n" +
			"var " + regexpVar(c.Param) + " = regexp.MustCompile(" + strconv.Quote(c.Param) + ")"
	},
	Emit: func(c vogue.EmitContext) string {
		return regexpVar(c.Param) + ".MatchString(" + c.Var + ")"
	},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: "^[A-Z]{3}-[0-9]{4}$", In: "SKU-0042", Note: "a code in the documented shape"},
			{Param: "^[A-Z]{3}-[0-9]{4}$", In: "EUR-1000", Note: "another code of the same shape"},
		},
		Invalid: []vogue.Example{
			{Param: "^[A-Z]{3}-[0-9]{4}$", In: "sku-0042", Note: "the prefix is not upper case"},
			{Param: "^[A-Z]{3}-[0-9]{4}$", In: "SKU-42", Note: "the number is too short"},
			{Param: "^[A-Z]{3}-[0-9]{4}$", In: "", Note: "the empty string, which an anchored pattern rejects"},
		},
	},
}

// OneOf restricts a value to a closed list.
var OneOf = vogue.Rule{
	Name:  "oneof",
	Kinds: vogue.Kinds(vogue.String, vogue.Int),
	Doc: "Restricts the value to one of the comma-separated items of the parameter, compared for " +
		"exact equality: on a string it is case-sensitive, so pair it with `lower` or `upper` when " +
		"the input is typed by a human, and on an integer every item must itself be an integer. " +
		"A list that names a set of states is usually better expressed as an `enum` directive, " +
		"which gives each member its own value and a Values accessor; this rule is for a closed " +
		"list that stays a plain string or number.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamList},
	Message: "{{.Field}} must be one of: {{.Param}}",
	Emit:    anyOf,
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Kinds: str, Param: "eur,usd,gbp", In: "eur", Note: "the first item of the list"},
			{Kinds: str, Param: "eur,usd,gbp", In: "gbp", Note: "the last item of the list"},
			{Kinds: intK, Param: "1,2,4", In: "1", Note: "the first allowed number"},
			{Kinds: intK, Param: "1,2,4", In: "4", Note: "the last allowed number"},
		},
		Invalid: []vogue.Example{
			{Kinds: str, Param: "eur,usd,gbp", In: "chf", Note: "a currency nobody listed"},
			{Kinds: str, Param: "eur,usd,gbp", In: "EUR", Note: "the right item in the wrong case"},
			{Kinds: intK, Param: "1,2,4", In: "3", Note: "a number between two allowed ones"},
			{Kinds: intK, Param: "1,2,4", In: "0", Note: "a number below the whole list"},
		},
	},
}

// Alpha allows letters only.
var Alpha = vogue.Rule{
	Name:  "alpha",
	Kinds: str,
	Doc: "Requires every rune of the value to be a letter, in the Unicode sense: \"Muñoz\" passes " +
		"and so does a name in Greek or Cyrillic, while a digit, a space, a hyphen or an " +
		"apostrophe does not. That makes it the wrong rule for a person's full name, which " +
		"routinely carries all three. The empty string passes, because it holds no offending " +
		"rune; pair it with `required` when the field is mandatory.",
	Message: "{{.Field}} must contain letters only",
	Imports: []string{importStrings, importUnicode},
	Emit:    func(c vogue.EmitContext) string { return noRuneWhere(c, "!unicode.IsLetter(r)") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "Tortilla", Note: "plain letters"},
			{In: "Muñoz", Note: "an accented letter is still a letter"},
		},
		Invalid: []vogue.Example{
			{In: "Tab7", Note: "a digit among the letters"},
			{In: "de patatas", Note: "a space, which is not a letter"},
		},
	},
}

// Alphanum allows letters and digits only.
var Alphanum = vogue.Rule{
	Name:  "alphanum",
	Kinds: str,
	Doc: "Requires every rune of the value to be a letter or a digit, in the Unicode sense. It is " +
		"the rule for a handle, a slug without separators or a short code: no spaces, no " +
		"punctuation, nothing that needs escaping downstream. A hyphen and an underscore are " +
		"rejected, so a slug that uses one needs a `regex` instead. The empty string passes; pair " +
		"it with `required`.",
	Message: "{{.Field}} must contain letters and digits only",
	Imports: []string{importStrings, importUnicode},
	Emit: func(c vogue.EmitContext) string {
		return noRuneWhere(c, "!unicode.IsLetter(r) && !unicode.IsDigit(r)")
	},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "Tab7", Note: "letters and digits together"},
			{In: "sku0042", Note: "a code with no separator"},
		},
		Invalid: []vogue.Example{
			{In: "sku-0042", Note: "a hyphen, which is neither a letter nor a digit"},
			{In: "tab 7", Note: "a space"},
		},
	},
}

// Numeric allows ASCII digits only.
var Numeric = vogue.Rule{
	Name:  "numeric",
	Kinds: str,
	Doc: "Requires every rune of the value to be an ASCII digit, 0 to 9. Unlike `alpha` and " +
		"`alphanum` it is deliberately not Unicode-aware: a value carrying Arabic-Indic digits " +
		"would satisfy unicode.IsDigit and then fail to parse as a number, and a numeric string " +
		"exists to be parsed. A sign and a decimal point are rejected, so it describes a digit " +
		"string — a phone number, a document number — rather than a number; use the `int` kind for " +
		"one of those. The empty string passes; pair it with `required`.",
	Message: "{{.Field}} must contain digits only",
	Imports: []string{importStrings},
	Emit:    func(c vogue.EmitContext) string { return noRuneWhere(c, "r < '0' || r > '9'") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "0042", Note: "a digit string keeping its leading zero"},
			{In: "600123456", Note: "a phone number"},
		},
		Invalid: []vogue.Example{
			{In: "-42", Note: "a sign, which a digit string does not carry"},
			{In: "4.2", Note: "a decimal point"},
			{In: "٤٢", Note: "Arabic-Indic digits, which no parser here would read"},
		},
	},
}

// ASCII allows ASCII runes only.
var ASCII = vogue.Rule{
	Name:  "ascii",
	Kinds: str,
	Doc: "Requires every rune of the value to be ASCII, below U+0080. It is the rule for a value " +
		"that has to survive a protocol or a legacy system that only speaks ASCII — a header, a " +
		"filename in an old archive format. It accepts control characters, since those are ASCII " +
		"too; combine it with `printable` when the value must also be readable. The empty string " +
		"passes; pair it with `required`.",
	Message: "{{.Field}} must contain ASCII characters only",
	Imports: []string{importStrings, importUTF8},
	Emit:    func(c vogue.EmitContext) string { return noRuneWhere(c, "r >= utf8.RuneSelf") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "tortilla", Note: "plain ASCII letters"},
			{In: "SKU-0042!", Note: "ASCII punctuation and digits"},
		},
		Invalid: []vogue.Example{
			{In: "Muñoz", Note: "an accented letter above U+007F"},
			{In: "café", Note: "a combining accent, which is not ASCII either"},
		},
	},
}

// Printable allows printable runes only.
var Printable = vogue.Rule{
	Name:  "printable",
	Kinds: str,
	Doc: "Requires every rune of the value to be printable as Go defines it: letters, marks, " +
		"numbers, punctuation, symbols and the ASCII space. Control characters are rejected, " +
		"including the newline and the tab, which is what keeps a single-line value from carrying " +
		"a line break into a log, a CSV export or a terminal. Accented and non-Latin letters pass, " +
		"so it restricts nothing a human would type into one line. The empty string passes; pair " +
		"it with `required`.",
	Message: "{{.Field}} must not contain control characters",
	Imports: []string{importStrings, importUnicode},
	Emit:    func(c vogue.EmitContext) string { return noRuneWhere(c, "!unicode.IsPrint(r)") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "Tortilla de patatas", Note: "an ordinary line of text"},
			{In: "Muñoz — 42 €", Note: "accents, a dash and a symbol are all printable"},
		},
		Invalid: []vogue.Example{
			{In: "Tortilla\nde patatas", Note: "a newline smuggled into a single-line value"},
			{In: "Tab\t7", Note: "a tab, which is a control character too"},
		},
	},
}

// NoSpace rejects whitespace anywhere in the value.
var NoSpace = vogue.Rule{
	Name:  "nospace",
	Kinds: str,
	Doc: "Rejects any whitespace anywhere in the value, in the Unicode sense: spaces, tabs, " +
		"newlines and the exotic blanks a paste carries. It is the rule for a token, a slug or a " +
		"code that has to appear unquoted in a URL, a header or a command line. It is a check, not " +
		"a normalizer: it reports the problem rather than silently removing it, which is what " +
		"`trim` and `squish` are for. The empty string passes; pair it with `required`.",
	Message: "{{.Field}} must not contain spaces",
	Imports: []string{importStrings, importUnicode},
	Emit:    func(c vogue.EmitContext) string { return "strings.IndexFunc(" + c.Var + ", unicode.IsSpace) < 0" },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "tortilla-de-patatas", Note: "a slug held together by hyphens"},
			{In: "SKU0042", Note: "a code with nothing to separate"},
		},
		Invalid: []vogue.Example{
			{In: "tortilla de patatas", Note: "a space in the middle"},
			{In: "sku0042 ", Note: "a trailing space a paste left behind"},
		},
	},
}

// Prefix requires the value to start with the parameter.
var Prefix = vogue.Rule{
	Name:  "prefix",
	Kinds: str,
	Doc: "Requires the value to start with the parameter, compared byte for byte and therefore " +
		"case-sensitively. It is the rule for an identifier whose namespace is part of its " +
		"meaning, such as \"SKU-\" or \"cus_\". The parameter may not contain a space, because a " +
		"directive is read as whitespace-separated tokens. An empty parameter accepts everything, " +
		"including the empty string.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamString},
	Message: `{{.Field}} must start with "{{.Param}}"`,
	Imports: []string{importStrings},
	Emit: func(c vogue.EmitContext) string {
		return "strings.HasPrefix(" + c.Var + ", " + strconv.Quote(c.Param) + ")"
	},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: "SKU-", In: "SKU-0042", Note: "the namespace the field is documented with"},
			{Param: "SKU-", In: "SKU-", Note: "the prefix alone, with nothing after it"},
		},
		Invalid: []vogue.Example{
			{Param: "SKU-", In: "sku-0042", Note: "the right prefix in the wrong case"},
			{Param: "SKU-", In: "0042", Note: "no namespace at all"},
			{Param: "SKU-", In: "", Note: "the empty string, which starts with nothing"},
		},
	},
}

// Suffix requires the value to end with the parameter.
var Suffix = vogue.Rule{
	Name:  "suffix",
	Kinds: str,
	Doc: "Requires the value to end with the parameter, compared byte for byte and therefore " +
		"case-sensitively. It is the rule for a value whose tail carries meaning, such as a file " +
		"extension or a domain. The parameter may not contain a space. An empty parameter accepts " +
		"everything, including the empty string.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamString},
	Message: `{{.Field}} must end with "{{.Param}}"`,
	Imports: []string{importStrings},
	Emit: func(c vogue.EmitContext) string {
		return "strings.HasSuffix(" + c.Var + ", " + strconv.Quote(c.Param) + ")"
	},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: ".pdf", In: "invoice.pdf", Note: "the extension the field expects"},
			{Param: ".pdf", In: ".pdf", Note: "the suffix alone, with nothing before it"},
		},
		Invalid: []vogue.Example{
			{Param: ".pdf", In: "invoice.PDF", Note: "the right extension in the wrong case"},
			{Param: ".pdf", In: "invoice.png", Note: "a different extension"},
		},
	},
}

// Contains requires the parameter to appear somewhere in the value.
var Contains = vogue.Rule{
	Name:  "contains",
	Kinds: str,
	Doc: "Requires the parameter to appear somewhere in the value, compared byte for byte and " +
		"therefore case-sensitively. It is a blunt rule, useful for a required separator — an \"@\", " +
		"a \"/\" — and a poor substitute for a shape: a value that must look like something is " +
		"better described by `regex` or by a named rule. The parameter may not contain a space. " +
		"An empty parameter accepts everything.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamString},
	Message: `{{.Field}} must contain "{{.Param}}"`,
	Imports: []string{importStrings},
	Emit: func(c vogue.EmitContext) string {
		return "strings.Contains(" + c.Var + ", " + strconv.Quote(c.Param) + ")"
	},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: "/", In: "tabs/7", Note: "the separator appears in the middle"},
			{Param: "/", In: "/", Note: "the separator is the whole value"},
		},
		Invalid: []vogue.Example{
			{Param: "/", In: "tabs-7", Note: "a different separator"},
			{Param: "/", In: "", Note: "the empty string, which contains nothing"},
		},
	},
}

// Excludes rejects a value in which the parameter appears.
var Excludes = vogue.Rule{
	Name:  "excludes",
	Kinds: str,
	Doc: "Rejects the value when the parameter appears anywhere in it, compared byte for byte and " +
		"therefore case-sensitively — which is exactly why it is a weak guard: a denylist is " +
		"defeated by a change of case or an encoding. Use it to keep a known separator out of a " +
		"field, never as a security control; for that, describe what is allowed with `regex` or " +
		"`alphanum` instead. The parameter may not contain a space, and an empty parameter would " +
		"reject everything.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamString},
	Message: `{{.Field}} must not contain "{{.Param}}"`,
	Imports: []string{importStrings},
	Emit: func(c vogue.EmitContext) string {
		return "!strings.Contains(" + c.Var + ", " + strconv.Quote(c.Param) + ")"
	},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: "/", In: "tabs-7", Note: "no forbidden separator anywhere"},
			{Param: "/", In: "", Note: "the empty string contains nothing to forbid"},
		},
		Invalid: []vogue.Example{
			{Param: "/", In: "tabs/7", Note: "the forbidden separator in the middle"},
			{Param: "/", In: "/tabs", Note: "the forbidden separator at the front"},
		},
	},
}
