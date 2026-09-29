package rules

import "github.com/MathiasHilgert/vogue"

// intK is the set of kinds an integer-only rule applies to.
var intK = vogue.Kinds(vogue.Int)

// decK is the set of kinds a decimal-only rule applies to.
var decK = vogue.Kinds(vogue.Decimal)

// numK is the set of kinds a rule about the sign of a number applies to.
var numK = vogue.Kinds(vogue.Int, vogue.Decimal)

// Positive requires a value strictly above zero.
var Positive = vogue.Rule{
	Name:  "positive",
	Kinds: numK,
	Doc: "Requires the value to be strictly greater than zero. It is the rule for a count or a " +
		"quantity that only means something when there is at least some of the thing: a party at " +
		"a table, a line on an order, a weight on a scale. Zero is rejected; when zero is a " +
		"legitimate value use `nonneg`, and when the floor is something else use `min`.",
	Message: "must be greater than zero",
	Emit:    func(c vogue.EmitContext) string { return signCompare(c, ">") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Kinds: intK, In: "1", Note: "the smallest positive count"},
			{Kinds: intK, In: "42", Note: "an ordinary count"},
			{Kinds: decK, In: "1.5", Note: "a kilo and a half on the scale"},
			{Kinds: decK, In: "0.001", Note: "a gram is still something"},
		},
		Invalid: []vogue.Example{
			{Kinds: intK, In: "0", Note: "zero, which is not positive"},
			{Kinds: intK, In: "-1", Note: "a negative count"},
			{Kinds: decK, In: "0.000", Note: "zero written at three decimal places is still zero"},
			{Kinds: decK, In: "-0.5", Note: "half a unit less than nothing"},
		},
	},
}

// NonNeg requires a value of zero or above.
var NonNeg = vogue.Rule{
	Name:  "nonneg",
	Kinds: numK,
	Doc: "Requires the value to be zero or greater. It is the rule for a quantity that may " +
		"legitimately be nothing — a stock level, a discount, a number of no-shows — where zero is " +
		"a real answer and a negative number is a mistake. It is `min=0` said in a way the " +
		"directive can be read out loud.",
	Message: "must not be negative",
	Emit:    func(c vogue.EmitContext) string { return signCompare(c, ">=") },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Kinds: intK, In: "0", Note: "nothing in stock is still an answer"},
			{Kinds: intK, In: "7", Note: "an ordinary quantity"},
			{Kinds: decK, In: "0", Note: "an empty shelf weighs nothing"},
			{Kinds: decK, In: "12.750", Note: "an ordinary weight"},
		},
		Invalid: []vogue.Example{
			{Kinds: intK, In: "-1", Note: "one below the floor"},
			{Kinds: intK, In: "-100", Note: "a quantity nobody can have"},
			{Kinds: decK, In: "-0.01", Note: "a hundredth below the floor"},
			{Kinds: decK, In: "-100", Note: "a quantity nobody can have"},
		},
	},
}

// MultipleOf requires the value to be a multiple of the parameter.
var MultipleOf = vogue.Rule{
	Name:  "multipleof",
	Kinds: intK,
	Doc: "Requires the value to be an exact multiple of the parameter, which is how a quantity " +
		"sold by the box, a duration measured in whole slots or an amount in whole units is " +
		"expressed. Negative values are accepted when they divide exactly, since -30 is as much a " +
		"multiple of 15 as 30 is; add `positive` or `nonneg` when they should not be. " +
		"`multipleof=0` accepts only zero, which is the mathematically honest reading of it.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
	Message: "must be a multiple of {{.Param}}",
	Emit: func(c vogue.EmitContext) string {
		// A modulo by a constant zero does not compile, so the degenerate
		// parameter is answered directly: zero is the only multiple of zero.
		if integer(c.Param) == "0" {
			return c.Var + " == 0"
		}
		return c.Var + "%" + c.Ident + " == 0"
	},
	Local: func(c vogue.EmitContext) string {
		if integer(c.Param) == "0" {
			return ""
		}
		return boundConst(c)
	},
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: "15", In: "30", Note: "two whole slots"},
			{Param: "15", In: "0", Note: "zero divides by everything"},
			{Param: "15", In: "-15", Note: "a negative multiple still divides exactly"},
		},
		Invalid: []vogue.Example{
			{Param: "15", In: "20", Note: "a duration that does not fill whole slots"},
			{Param: "15", In: "1", Note: "less than one slot"},
		},
	},
}

// Scale bounds the number of decimal places a decimal value carries.
var Scale = vogue.Rule{
	Name:  "scale",
	Kinds: decK,
	Doc: "Requires the value to carry at most the given number of decimal places, which is how a " +
		"rate stored in a numeric(p,s) column, a unit price quoted to the cent or a weight " +
		"measured to the gram is expressed. It rejects a value that is too precise, it does not " +
		"round one: rounding loses information, and how to lose it — up, down, half to even, at " +
		"which point in the calculation — is a decision the domain makes rather than the " +
		"constructor. Round before calling the constructor and this rule will tell you when you " +
		"forgot. The scale compared is the scale of the value as written, so \"0.5000\" carries " +
		"four decimal places even though \"0.5\" is the same number.",
	Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
	Message: "must have at most {{.Param}} decimal places",
	Emit:    func(c vogue.EmitContext) string { return c.Var + ".Scale() <= " + c.Ident },
	Local:   func(c vogue.EmitContext) string { return "const " + c.Ident + " = " + integer(c.Param) },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{Param: "4", In: "0.1234", Note: "exactly the places the bound allows"},
			{Param: "4", In: "0.5", Note: "fewer places than the bound allows"},
			{Param: "3", In: "1.5", Note: "one place fits in three"},
			{Param: "3", In: "12.750", Note: "three places are exactly three"},
		},
		Invalid: []vogue.Example{
			{Param: "4", In: "0.12345", Note: "one decimal place more than the column holds"},
			{Param: "3", In: "0.1234", Note: "a weight measured finer than the scale reads"},
		},
	},
}

// NonZero rejects a decimal that is zero however it is written.
var NonZero = vogue.Rule{
	Name:  "nonzero",
	Kinds: decK,
	Doc: "Rejects zero, at any scale: \"0\", \"0.00\" and \"-0.0\" are all the same number and all " +
		"rejected. It is the rule for a value that is meaningless when it is nothing — an " +
		"adjustment that does not adjust, a divisor, a conversion rate — and it says so without " +
		"taking a side on the sign, which `positive` and `nonneg` do. Note that a value object " +
		"holding zero is indistinguishable from the zero value of its type, so a field that may " +
		"legitimately be zero is better modelled as a pointer or an optional than talked out of " +
		"it by a rule.",
	Message: "must not be zero",
	Emit:    func(c vogue.EmitContext) string { return "!" + c.Var + ".IsZero()" },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "1", Note: "a whole unit"},
			{In: "-0.5", Note: "a negative adjustment still adjusts"},
		},
		Invalid: []vogue.Example{
			{In: "0", Note: "nothing to adjust"},
			{In: "0.00", Note: "nothing, written to the cent"},
		},
	},
}
