package rules

import (
	"strconv"
	"strings"

	"github.com/MathiasHilgert/vogue"
)

// Method names of the built-in rules that are written as an unexported method
// of the generated type, named once so a rule and its tests cannot drift apart.
const (
	methodEmail    = "isEmail"
	methodURL      = "isURL"
	methodUUID     = "isUUID"
	methodTimeZone = "isTimeZone"
)

// maxCaseLine is the widest a line of zone names grows, tabs counted as four
// columns, so the generated switch stays readable and under a line-length
// linter's limit.
const maxCaseLine = 100

// body joins the lines of a method body. Every line is written with the tabs
// it has inside the method, which gofmt keeps.
func body(lines ...string) string { return strings.Join(lines, "\n") }

// emailMethod is the body of the email check: a bare RFC 5322 address parsed
// by net/mail, with no display name and nothing around it.
func emailMethod(c vogue.EmitContext) (name, source string) {
	return methodEmail, body(
		"\taddress, err := mail.ParseAddress("+c.Var+")",
		"",
		"\treturn err == nil && address.Name == \"\" && address.Address == "+c.Var,
	)
}

// urlMethod is the body of the url check: an absolute http or https URL with
// a host, parsed by net/url as a request URI.
func urlMethod(c vogue.EmitContext) (name, source string) {
	return methodURL, body(
		"\tparsed, err := url.ParseRequestURI("+c.Var+")",
		"\tif err != nil || parsed.Host == \"\" {",
		"\t\treturn false",
		"\t}",
		"",
		"\treturn parsed.Scheme == \"http\" || parsed.Scheme == \"https\"",
	)
}

// uuidMethod is the body of the uuid check: the canonical 8-4-4-4-12 form,
// hexadecimal digits of either case with hyphens where the layout has them.
func uuidMethod(c vogue.EmitContext) (name, source string) {
	return methodUUID, body(
		"\tconst (",
		"\t\tlayout    = \"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx\"",
		"\t\thexDigits = \"0123456789abcdefABCDEF\"",
		"\t)",
		"",
		"\tif len("+c.Var+") != len(layout) {",
		"\t\treturn false",
		"\t}",
		"",
		"\tfor index := range len(layout) {",
		"\t\tallowed := hexDigits",
		"\t\tif layout[index] == '-' {",
		"\t\t\tallowed = \"-\"",
		"\t\t}",
		"",
		"\t\tif strings.IndexByte(allowed, "+c.Var+"[index]) < 0 {",
		"\t\t\treturn false",
		"\t\t}",
		"\t}",
		"",
		"\treturn true",
	)
}

// timeZoneMethod is the body of the timezone check: a switch over every zone
// name of the IANA database, several names to a line. A switch needs no zone
// database at run time and compares case-exactly, which time.LoadLocation does
// not on a case-insensitive file system.
func timeZoneMethod(c vogue.EmitContext) (name, source string) {
	return methodTimeZone, body(
		"\tswitch "+c.Var+" {",
		"\tcase "+packedCase(timeZoneNames[:])+":",
		"\t\treturn true",
		"\t}",
		"",
		"\treturn false",
	)
}

// packedCase renders names as the expression list of one case clause, wrapped
// so that no line is wider than [maxCaseLine].
func packedCase(names []string) string {
	const (
		firstIndent = len("\tcase ")
		nextIndent  = len("\t\t") * 2
		tabWidth    = 4
	)
	var (
		out   strings.Builder
		width = firstIndent - 1 + tabWidth
	)
	for i, name := range names {
		quoted := strconv.Quote(name)
		if i > 0 {
			out.WriteString(",")
			if width+len(" ")+len(quoted)+len(",") > maxCaseLine {
				out.WriteString("\n\t\t")
				width = nextIndent
			} else {
				out.WriteString(" ")
				width += len(", ")
			}
		}
		out.WriteString(quoted)
		width += len(quoted)
	}
	return out.String()
}
