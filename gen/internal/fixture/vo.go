// Package fixture holds the directives the generator is proven against. Its
// generated counterpart is committed next to this file and compiled by the
// ordinary test run, so `go test ./...` fails the moment the generator starts
// producing code that does not compile or does not behave.
package fixture

import "strings"

// NoDigits reports whether v contains no decimal digit. It is the target of
// the call-backed rule, declared in this package so the generator is exercised
// on a rule whose function lives in the package being generated into.
func NoDigits(v string) bool {
	return !strings.ContainsAny(v, "0123456789")
}

// Title is the name of a tab, as the waiter typed it. It is trimmed and folded
// to lower case before its bounds are checked.
//vogue:string Title required trim lower min=1 max=120 nodigits

// Covers is the number of guests seated at a tab.
//vogue:int Covers min=1 max=200

// TabStatus is the lifecycle state of a tab.
//vogue:enum TabStatus open,in_progress,closed

// TabID identifies a tab across services.
//vogue:id TabID uuid7

// InvoiceNumber is the sequence the accounting system assigns to an invoice.
//vogue:id InvoiceNumber int64
