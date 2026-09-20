package rules

import "github.com/MathiasHilgert/vogue"

// intK is the set of kinds an integer-only rule applies to.
var intK = vogue.Kinds(vogue.Int)

// Positive requires a value strictly above zero.
var Positive = vogue.Rule{
	Name:  "positive",
	Kinds: intK,
	Doc: "Requires the value to be strictly greater than zero. It is the rule for a count that " +
		"only means something when there is at least one of the thing: a party at a table, a line " +
		"on an order. Zero is rejected; when zero is a legitimate value use `nonneg`, and when the " +
		"floor is something else use `min`.",
	Message: "{{.Field}} must be greater than zero",
	Emit:    func(c vogue.EmitContext) string { return c.Var + " > 0" },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "1", Note: "the smallest positive count"},
			{In: "42", Note: "an ordinary count"},
		},
		Invalid: []vogue.Example{
			{In: "0", Note: "zero, which is not positive"},
			{In: "-1", Note: "a negative count"},
		},
	},
}

// NonNeg requires a value of zero or above.
var NonNeg = vogue.Rule{
	Name:  "nonneg",
	Kinds: intK,
	Doc: "Requires the value to be zero or greater. It is the rule for a quantity that may " +
		"legitimately be nothing — a stock level, a discount, a number of no-shows — where zero is " +
		"a real answer and a negative number is a mistake. It is `min=0` said in a way the " +
		"directive can be read out loud.",
	Message: "{{.Field}} must not be negative",
	Emit:    func(c vogue.EmitContext) string { return c.Var + " >= 0" },
	Examples: vogue.Examples{
		Valid: []vogue.Example{
			{In: "0", Note: "nothing in stock is still an answer"},
			{In: "7", Note: "an ordinary quantity"},
		},
		Invalid: []vogue.Example{
			{In: "-1", Note: "one below the floor"},
			{In: "-100", Note: "a quantity nobody can have"},
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
	Message: "{{.Field}} must be a multiple of {{.Param}}",
	Emit: func(c vogue.EmitContext) string {
		// A modulo by a constant zero does not compile, so the degenerate
		// parameter is answered directly: zero is the only multiple of zero.
		if c.Param == "0" {
			return c.Var + " == 0"
		}
		return c.Var + "%" + c.Param + " == 0"
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
