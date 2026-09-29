package rules

import (
	"math"
	"strconv"
	"strings"

	"github.com/govalues/decimal"

	"github.com/MathiasHilgert/vogue"
)

// Import paths the emitted expressions need, named once so an expression and
// the import it depends on cannot drift apart.
const (
	importDecimal = "github.com/govalues/decimal"
	importMail    = "net/mail"
	importRegexp  = "regexp"
	importURL     = "net/url"
	importSlices  = "slices"
	importStrings = "strings"
	importUnicode = "unicode"
	importUTF8    = "unicode/utf8"
)

// runeCount returns the expression counting the runes of the working value.
func runeCount(c vogue.EmitContext) string {
	return "utf8.RuneCountInString(" + c.Var + ")"
}

// noRuneWhere returns an expression that is true when no rune of the working
// value satisfies pred, which is the shape every character-class check has.
// The empty string satisfies it, since it holds no offending rune.
func noRuneWhere(c vogue.EmitContext, pred string) string {
	return "strings.IndexFunc(" + c.Var + ", func(character rune) bool { return " + pred + " }) < 0"
}

// compare returns the comparison a bound rule emits for the kind being
// generated: a rune count for a string, the value itself for an integer, and a
// three-way Cmp for a decimal. The bound is never written inline: it is the
// constant [boundConst] declares, so the constructor reads `value <= maximumParameter`
// rather than comparing against a bare number.
func compare(c vogue.EmitContext, op string) string {
	switch c.Kind {
	case vogue.Int:
		return c.Var + " " + op + " " + c.Ident
	case vogue.Decimal:
		_, scale, ok := decimalParts(c.Param)
		switch {
		case !ok:
			return c.Var + ".Cmp(decimal.MustParse(" + c.Ident + ")) " + op + " 0"
		case scale == 0:
			return c.Var + ".Cmp(decimal.MustNew(" + c.Ident + ", 0)) " + op + " 0"
		default:
			return c.Var + ".Cmp(decimal.MustNew(" + c.Ident + "Coef, " + c.Ident + "Scale)) " + op + " 0"
		}
	default:
		return runeCount(c) + " " + op + " " + c.Ident
	}
}

// boundConst declares, inside the constructor, the constant a bound rule
// compares against.
//
// A decimal bound is declared as the coefficient and scale
// decimal.MustNew takes — the coefficient alone when the bound is a whole
// number — so the constructor builds the bound without parsing text; both were
// validated at generate time, so MustNew cannot panic. A
// bound whose coefficient does not fit an int64 is declared as its text and
// parsed instead, which is correct if slower and never happens for a bound a
// person writes.
func boundConst(c vogue.EmitContext) string {
	if c.Kind != vogue.Decimal {
		return "const " + c.Ident + " = " + integer(c.Param)
	}
	coef, scale, ok := decimalParts(c.Param)
	switch {
	case !ok:
		return "const " + c.Ident + " = " + strconv.Quote(c.Param)
	case scale == 0:
		return "const " + c.Ident + " = " + strconv.FormatInt(coef, 10)
	default:
		return "const " + c.Ident + "Coef = " + strconv.FormatInt(coef, 10) + "\n" +
			"const " + c.Ident + "Scale = " + strconv.Itoa(scale)
	}
}

// decimalParts returns the signed coefficient and the scale of a decimal
// bound, and false when the coefficient does not fit an int64.
func decimalParts(param string) (coef int64, scale int, ok bool) {
	d, err := decimal.Parse(param)
	if err != nil || d.Coef() > math.MaxInt64 {
		return 0, 0, false
	}
	coef = int64(d.Coef()) //nolint:gosec // bounded by the check above.
	if d.Sign() < 0 {
		coef = -coef
	}
	return coef, d.Scale(), true
}

// signCompare returns the sign test a sign rule emits for the kind being
// generated.
func signCompare(c vogue.EmitContext, op string) string {
	if c.Kind == vogue.Decimal {
		return c.Var + ".Sign() " + op + " 0"
	}
	return c.Var + " " + op + " 0"
}

// anyOf returns the disjunction a membership rule emits: every item of the
// comma-separated parameter compared against the working value. String items
// are quoted, integer items are emitted as written, because the working value
// is an int64 there.
//
// On an integer the list is a slices.Contains over a literal slice instead: a
// chain of `v == 4` comparisons is a chain of magic numbers to a linter, while
// the items of a literal are plainly data.
func anyOf(c vogue.EmitContext) string {
	items := strings.Split(c.Param, ",")
	if c.Kind == vogue.Int {
		for i, item := range items {
			items[i] = integer(item)
		}
		return "slices.Contains([]int64{" + strings.Join(items, ", ") + "}, " + c.Var + ")"
	}
	terms := make([]string, len(items))
	for i, item := range items {
		terms[i] = c.Var + " == " + strconv.Quote(item)
	}
	return strings.Join(terms, " || ")
}

// integer returns an integer parameter the way Go source must spell it: the
// base-10 number it parses to. The directive is read in base 10, but Go reads
// a literal with a leading zero in octal, so `min=010` written through would
// compare against 8. A parameter that does not parse is returned unchanged;
// the parser has already rejected it.
func integer(param string) string {
	parsed, err := strconv.ParseInt(param, 10, 64)
	if err != nil {
		return param
	}
	return strconv.FormatInt(parsed, 10)
}

// patternName returns the name of the package-level variable holding the
// compiled pattern of a regex rule: the receiver of the type, which is its
// name in lower camel case, followed by "Pattern". A directive uses a rule
// once, so one type never declares two of them.
func patternName(c vogue.EmitContext) string { return c.Receiver + "Pattern" }

// regexDeclaration declares the compiled pattern of a regex rule, once per
// generated type, at package initialisation. The generator has already
// compiled the pattern to validate the directive, so MustCompile cannot panic
// at run time. A raw string keeps the pattern as it was written, unless it
// holds a backtick, which a raw string cannot.
func regexDeclaration(c vogue.EmitContext) string {
	literal := "`" + c.Param + "`"
	if strings.Contains(c.Param, "`") {
		literal = strconv.Quote(c.Param)
	}
	return "var " + patternName(c) + " = regexp.MustCompile(" + literal + ")"
}
