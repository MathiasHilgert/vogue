package vogue_test

import (
	"errors"
	"fmt"

	"github.com/MathiasHilgert/vogue"
)

// ExampleFieldError shows the canonical rendering of a single rule failure and
// its machine-readable code.
func ExampleFieldError() {
	err := vogue.FieldError{
		Field:   "title",
		Rule:    "min",
		Param:   "1",
		Value:   "",
		Message: "must be between 1 and 120 characters",
	}

	fmt.Println(err.Error())
	fmt.Println(err.Code())
	// Output:
	// title: must be between 1 and 120 characters (rule "min", param "1")
	// title.min
}

// ExampleNotification shows the Notification pattern a generated constructor
// follows: collect every failure, then return them as one error.
func ExampleNotification() {
	newTitle := func(raw string) (string, error) {
		var n vogue.Notification
		if raw == "" {
			n.Addf("title", "required", "", raw, "is required")
		}
		if len(raw) > 5 {
			n.Addf("title", "max", "5", raw, "must be at most %d characters", 5)
		}
		return raw, n.ErrOrNil()
	}

	_, err := newTitle("")
	fmt.Println(err)

	_, err = newTitle("a valid but overly long title")
	fmt.Println(err)

	_, err = newTitle("ok")
	fmt.Println(err)
	// Output:
	// 1 validation error:
	//   - title: is required (rule "required")
	// 1 validation error:
	//   - title: must be at most 5 characters (rule "max", param "5")
	// <nil>
}

// ExampleNotification_errorsAs shows that a Notification cooperates with the
// standard errors package: individual failures are reachable with errors.Is
// and errors.As without unwrapping the concrete type by hand.
func ExampleNotification_errorsAs() {
	var n vogue.Notification
	n.Addf("title", "min", "1", "", "is too short")
	n.Addf("email", "email", "", "nope", "must be a valid email address")
	err := n.ErrOrNil()

	fmt.Println(errors.Is(err, vogue.FieldError{Rule: "email"}))
	fmt.Println(errors.Is(err, vogue.FieldError{Rule: "uuid"}))

	var first vogue.FieldError
	if errors.As(err, &first) {
		fmt.Println(first.Code())
	}
	// Output:
	// true
	// false
	// title.min
}

// ExampleNotification_Field shows how to pick out the failures of one field,
// which is what an HTTP layer needs to build a per-field error payload.
func ExampleNotification_Field() {
	var n vogue.Notification
	n.Addf("title", "required", "", "", "is required")
	n.Addf("email", "email", "", "nope", "must be a valid email address")
	n.Addf("title", "min", "1", "", "is too short")

	for _, e := range n.Field("title") {
		fmt.Println(e.Code())
	}
	// Output:
	// title.required
	// title.min
}

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
