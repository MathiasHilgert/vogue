package rules

import "github.com/MathiasHilgert/vogue"

// Trim removes the whitespace around a value before the rules written after it
// run.
var Trim = vogue.Rule{
	Name:  "trim",
	Kinds: vogue.Kinds(vogue.String),
	Doc: "Removes leading and trailing whitespace, in the Unicode sense: spaces, tabs, " +
		"newlines and the exotic blanks a paste from a word processor carries. It rewrites " +
		"the value rather than rejecting it, so every rule written after it measures the " +
		"trimmed value; a field that must not be blank pairs it with `required`, which then " +
		"rejects a value that was nothing but whitespace.",
	Message:   "is trimmed",
	Normalize: true,
	Imports:   []string{importStrings},
	Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.TrimSpace(" + c.Var + ")" },
	Examples: vogue.Examples{
		Normalized: []vogue.Normalization{
			{In: "  Tortilla  ", Out: "Tortilla", Note: "the blanks around a pasted value are dropped"},
			{In: "\tTortilla\n", Out: "Tortilla", Note: "tabs and newlines count as whitespace too"},
			{In: "Tortilla", Out: "Tortilla", Note: "an already clean value is left alone"},
		},
	},
}

// Squish collapses runs of internal whitespace to a single space and trims the
// ends.
var Squish = vogue.Rule{
	Name:  "squish",
	Kinds: vogue.Kinds(vogue.String),
	Doc: "Collapses every run of whitespace inside the value to one space and removes the " +
		"whitespace around it, so \"  Tortilla   de   patatas \" becomes \"Tortilla de patatas\". " +
		"It is the normalizer for a value a human typed into a single-line field, where a double " +
		"space is a slip rather than a meaning. Line breaks are collapsed as well, which makes it " +
		"the wrong rule for a value whose layout matters.",
	Message:   "has its whitespace squished",
	Normalize: true,
	Imports:   []string{importStrings},
	Emit: func(c vogue.EmitContext) string {
		return c.Var + " = strings.Join(strings.Fields(" + c.Var + "), \" \")"
	},
	Examples: vogue.Examples{
		Normalized: []vogue.Normalization{
			{In: "Tortilla   de  patatas", Out: "Tortilla de patatas", Note: "a run of spaces becomes one"},
			{In: "  Tortilla de patatas  ", Out: "Tortilla de patatas", Note: "the ends are trimmed as well"},
			{In: "Tortilla\tde\npatatas", Out: "Tortilla de patatas", Note: "a tab and a newline become plain spaces"},
		},
	},
}

// Lower folds a value to lower case before the rules written after it run.
var Lower = vogue.Rule{
	Name:  "lower",
	Kinds: vogue.Kinds(vogue.String),
	Doc: "Folds the value to lower case using the Unicode mapping, so \"Í\" becomes \"í\". " +
		"It is how a value that must compare case-insensitively — an email address, a slug, a " +
		"tag — is stored in one canonical shape, which makes an index on it meaningful. The " +
		"mapping is locale-independent, so the Turkish dotless i is not special-cased.",
	Message:   "is lower-cased",
	Normalize: true,
	Imports:   []string{importStrings},
	Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.ToLower(" + c.Var + ")" },
	Examples: vogue.Examples{
		Normalized: []vogue.Normalization{
			{In: "Waiter@Example.Com", Out: "waiter@example.com", Note: "an address is folded to one canonical spelling"},
			{In: "ÁRBOL", Out: "árbol", Note: "an accented capital folds like any other letter"},
			{In: "already", Out: "already", Note: "a value already in lower case is left alone"},
		},
	},
}

// Upper folds a value to upper case before the rules written after it run.
var Upper = vogue.Rule{
	Name:  "upper",
	Kinds: vogue.Kinds(vogue.String),
	Doc: "Folds the value to upper case using the Unicode mapping. It is the normalizer for a " +
		"code that is conventionally shouted — a currency code, a country code, a SKU prefix — " +
		"so the value reaches the database in the shape the rest of the system expects to read. " +
		"Like `lower`, the mapping is locale-independent.",
	Message:   "is upper-cased",
	Normalize: true,
	Imports:   []string{importStrings},
	Emit:      func(c vogue.EmitContext) string { return c.Var + " = strings.ToUpper(" + c.Var + ")" },
	Examples: vogue.Examples{
		Normalized: []vogue.Normalization{
			{In: "eur", Out: "EUR", Note: "a currency code is shouted the way the standard writes it"},
			{In: "sku-12", Out: "SKU-12", Note: "digits and punctuation are left untouched"},
			{In: "EUR", Out: "EUR", Note: "a value already in upper case is left alone"},
		},
	},
}
