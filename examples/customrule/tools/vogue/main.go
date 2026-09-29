// Command vogue is this example's own generator: the built-in catalogue plus
// the rules the project brings of its own.
//
// It is the whole of the extension mechanism. There is no plugin, no runtime
// registry and no configuration file — a project that needs a rule writes one
// and builds the eight lines below, which is what keeps the rule set statically
// known and lets an unknown tag be a compile-time-shaped error.
package main

import (
	"log"

	"github.com/MathiasHilgert/vogue/examples/customrule/cuitrule"
	"github.com/MathiasHilgert/vogue/generator"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("")
	if err := generator.Run(generator.WithRules(cuitrule.Rule)); err != nil {
		log.Fatal(err)
	}
}
