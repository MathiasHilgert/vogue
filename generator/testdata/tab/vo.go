// Package tab is the directive fixture the generator entry point is proven
// against. It lives under testdata, so the Go tool never builds it: every test
// copies it into a temporary directory and generates there.
package tab

// Title is the name of a tab, as the waiter typed it.
//vogue:string Title trim required min=1 max=120

// Covers is the number of guests seated at a tab.
//vogue:int Covers min=1 max=200
