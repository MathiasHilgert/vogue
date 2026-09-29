package vogue_test

import (
	"fmt"

	"github.com/MathiasHilgert/vogue"
)

// ExampleRuleSet_Suggest shows the "did you mean" lookup behind the
// unknown-rule diagnostic the generator reports with a file:line position.
func ExampleRuleSet_Suggest() {
	var set vogue.RuleSet
	for _, name := range []string{"min", "max", "email", "oneof"} {
		_ = set.Add(vogue.Rule{
			Name:    name,
			Kinds:   vogue.Kinds(vogue.String),
			Doc:     "Example rule.",
			Message: "{{.Field}} is invalid",
			Emit:    func(vogue.EmitContext) string { return "true" },
		})
	}

	if suggestion, ok := set.Suggest("emial"); ok {
		fmt.Printf("unknown rule %q, did you mean %q?\n", "emial", suggestion)
	}
	if _, ok := set.Suggest("completelydifferent"); !ok {
		fmt.Printf("known rules: %v\n", set.Names())
	}
	// Output:
	// unknown rule "emial", did you mean "email"?
	// known rules: [min max email oneof]
}
