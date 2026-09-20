package vogue_test

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/rules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The markers delimiting the generated catalogue table in the README. They are
// HTML comments, so they are invisible wherever the file is rendered.
const (
	readmePath  = "README.md"
	tableStart  = "<!-- rules:start -->"
	tableEnd    = "<!-- rules:end -->"
	tableHeader = "| Rule | Kinds | Param | Description |\n|------|-------|-------|-------------|"
)

// updateREADME rewrites the catalogue table instead of comparing against it,
// for when a rule is added or its first sentence changes.
var updateREADME = flag.Bool("update", false, "rewrite the rule table of the README")

// ruleTable renders the built-in catalogue as a Markdown table.
func ruleTable() string {
	var b strings.Builder
	b.WriteString(tableHeader)
	for _, rule := range rules.All() {
		fmt.Fprintf(&b, "\n| `%s` | %s | %s | %s |",
			rule.Name, rule.Kinds, rule.Param, escapePipes(rule.Summary()))
	}
	return b.String()
}

// escapePipes keeps a vertical bar in a rule's documentation from splitting the
// cell it is written in.
func escapePipes(s string) string { return strings.ReplaceAll(s, "|", `\|`) }

func TestREADME_RuleTable(t *testing.T) {
	t.Run("the table lists the catalogue as it stands today", func(t *testing.T) {
		// Arrange
		body, err := os.ReadFile(readmePath)
		require.NoError(t, err)
		readme := string(body)

		start := strings.Index(readme, tableStart)
		end := strings.Index(readme, tableEnd)
		require.GreaterOrEqual(t, start, 0, "the README is missing the %s marker", tableStart)
		require.Greater(t, end, start, "the README is missing the %s marker", tableEnd)

		want := tableStart + "\n" + ruleTable() + "\n" + tableEnd

		// Act
		if *updateREADME {
			fresh := readme[:start] + want + readme[end+len(tableEnd):]
			require.NoError(t, os.WriteFile(readmePath, []byte(fresh), 0o600))
			readme = fresh
		}

		// Assert
		assert.Contains(t, readme, want,
			"the rule table of the README has drifted; re-run `go test . -update`")
	})

	t.Run("every shipped rule has a row and a first sentence to put in it", func(t *testing.T) {
		// Arrange
		body, err := os.ReadFile(readmePath)
		require.NoError(t, err)
		readme := string(body)

		// Assert
		for _, rule := range rules.All() {
			assert.NotEmpty(t, rule.Summary(), "rule %q has no documentation to summarise", rule.Name)
			assert.Contains(t, readme, "| `"+rule.Name+"` |", "rule %q has no row in the README", rule.Name)
		}
	})

	t.Run("a rule table row names the kinds and the parameter contract", func(t *testing.T) {
		// Arrange
		min, ok := rules.MustSet().Get("min")
		require.True(t, ok)

		// Act
		table := ruleTable()

		// Assert
		assert.Contains(t, table, "| `min` | string, int, decimal | required number |")
		assert.Equal(t, vogue.Kinds(vogue.String, vogue.Int, vogue.Decimal), min.Kinds)
	})
}
