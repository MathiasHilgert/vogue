package gen

import (
	"fmt"
	"strings"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
)

// valueSentinel is substituted for the runtime value when a message is rendered
// at generate time. Finding it in the result means the template referenced
// {{.Value}}, which cannot be honoured by a message rendered before the value
// exists.
const valueSentinel = "\x00vogue-runtime-value\x00"

// fileView is the data the header template renders. Imports are split the way
// goimports groups them, standard library first, so the generated file looks
// like a file someone wrote.
type fileView struct {
	Package string
	Std     []string
	Ext     []string
	// Decls are the package-level declarations the rules of the file asked for,
	// emitted between the imports and the first value object.
	Decls []string
}

// stepView is one rule of a constructor, already resolved to Go source.
type stepView struct {
	// Normalize marks a step that rewrites the working value instead of
	// checking it, in which case Code is a statement and the rest is unused.
	Normalize bool
	// Code is the emitted statement, or the expression that is true when the
	// value is valid.
	Code string
	// Field, Rule and Param identify the failure the step records.
	Field, Rule, Param string
	// Message is the failure message, rendered at generate time.
	Message string
}

// memberView is one member of a generated enum.
type memberView struct {
	Const, Value string
}

// voView is the data a kind template renders. Fields that do not apply to the
// kind being rendered are left at their zero value.
type voView struct {
	// Name is the generated type name and Recv the receiver identifier used by
	// its methods.
	Name, Recv string
	// Field is the name the value object reports in a [vogue.FieldError].
	Field string
	// DocLines are the godoc lines of the type, without their slashes.
	DocLines []string
	// ValueExpr renders the offending value as text inside a constructor.
	ValueExpr string
	// Steps are the rules of a string or int constructor, in directive order.
	Steps []stepView
	// Members, MemberList and MemberParam describe an enum.
	Members     []memberView
	MemberList  string
	MemberParam string
	// Mint is the uuid constructor an id value object mints through, and
	// MintDoc the sentence documenting the strategy.
	Mint, MintDoc string
}

// newView builds the template data of one directive, names the template that
// renders it and records the imports it needs.
func (g *Generator) newView(d parse.Directive, imports *importSet, decls *declSet) (voView, string, error) {
	v := voView{
		Name:     d.Name,
		Recv:     receiver(d.Name),
		Field:    d.Field,
		DocLines: docLines(d),
	}

	// Every kind reports failures through vogue and formats an unsupported
	// Scan source with fmt; the rest depends on the shape being generated.
	paths := []string{importVogue, importFmt}

	var name string
	switch d.Kind {
	case vogue.String:
		v.ValueExpr, name = "v", "string"
		paths = append(paths, importDriver)

	case vogue.Int:
		v.ValueExpr, name = "strconv.FormatInt(v, 10)", "int"
		paths = append(paths, importDriver, importStrconv)

	case vogue.Decimal:
		v.ValueExpr, name = "v.String()", "decimal"
		paths = append(paths, importDriver, importDecimal)

	case vogue.Enum:
		name = "enum"
		paths = append(paths, importDriver)
		g.enumView(&v, d)

	case vogue.ID:
		name, paths = g.idView(&v, d, paths)

	default:
		return voView{}, "", fmt.Errorf("gen: %s: unsupported kind %s", d.Pos, d.Kind)
	}

	if err := addAll(imports, paths...); err != nil {
		return voView{}, "", err
	}
	if len(d.Rules) > 0 {
		steps, err := g.steps(d, imports, decls)
		if err != nil {
			return voView{}, "", err
		}
		v.Steps = steps
	}
	return v, name, nil
}

// enumView fills in the members of an enum.
func (g *Generator) enumView(v *voView, d parse.Directive) {
	values := make([]string, len(d.Values))
	v.Members = make([]memberView, len(d.Values))
	for i, member := range d.Values {
		v.Members[i] = memberView{Const: member.Const, Value: member.Value}
		values[i] = member.Value
	}
	v.MemberParam = strings.Join(values, ",")
	v.MemberList = strings.Join(values, ", ")
}

// idView completes the view of an id value object for its strategy, returning
// the template that renders it and the imports it adds.
func (g *Generator) idView(v *voView, d parse.Directive, paths []string) (string, []string) {
	if d.Strategy == parse.IDInt64 {
		v.ValueExpr = "strconv.FormatInt(v, 10)"
		return "id_int64", append(paths, importDriver, importStrconv)
	}

	if d.Strategy == parse.IDUUIDv4 {
		v.Mint, v.MintDoc = "uuid.NewRandom", "a random UUIDv4"
	} else {
		v.Mint, v.MintDoc = "uuid.NewV7", "a time-ordered UUIDv7, which keeps inserts index-friendly"
	}
	return "id_uuid", append(paths, importUUID)
}

// addUsed records the import paths of a rule that the source it just emitted
// actually references.
//
// [vogue.Rule.Imports] is declared once for the whole rule, but a rule
// spanning several kinds emits different source for each of them: `min` counts
// runes with unicode/utf8 on a string and compares a once-parsed
// github.com/govalues/decimal value on a decimal, and neither file may import
// what the other needs. Which of the two it is can only be read off the source
// the rule produced, so that is what is asked, by looking for the identifier
// the path is referenced through.
func addUsed(imports *importSet, paths []string, sources ...string) error {
	for _, path := range paths {
		selector := packageIdent(path) + "."
		used := false
		for _, source := range sources {
			if strings.Contains(source, selector) {
				used = true
				break
			}
		}
		if !used {
			continue
		}
		if err := imports.add(path); err != nil {
			return err
		}
	}
	return nil
}

// addAll records several import paths, stopping at the first collision.
func addAll(imports *importSet, paths ...string) error {
	for _, path := range paths {
		if err := imports.add(path); err != nil {
			return err
		}
	}
	return nil
}

// steps resolves every rule of a directive to the source the constructor runs,
// preserving the order the rules were written in.
func (g *Generator) steps(d parse.Directive, imports *importSet, decls *declSet) ([]stepView, error) {
	steps := make([]stepView, 0, len(d.Rules))
	for _, use := range d.Rules {
		rule := use.Rule

		var decl string
		if rule.Declare != nil {
			decl = rule.Declare(emitContext(d, use))
			decls.add(decl)
		}

		code, err := g.code(d, use, imports)
		if err != nil {
			return nil, err
		}
		if err := addUsed(imports, rule.Imports, code, decl); err != nil {
			return nil, err
		}
		if rule.Normalize {
			steps = append(steps, stepView{Normalize: true, Code: code})
			continue
		}

		message, err := renderMessage(rule, d.Field, use.Param)
		if err != nil {
			return nil, fmt.Errorf("gen: %s: %w", use.Pos, err)
		}
		steps = append(steps, stepView{
			Code:    code,
			Field:   d.Field,
			Rule:    rule.Name,
			Param:   use.Param,
			Message: message,
		})
	}
	return steps, nil
}

// code returns the Go source of one rule: the expression or statement the rule
// emits, or the static call it delegates to.
func (g *Generator) code(d parse.Directive, use parse.RuleUse, imports *importSet) (string, error) {
	rule := use.Rule
	if rule.Call != nil {
		if err := imports.add(rule.Call.Path); err != nil {
			return "", err
		}
		call := rule.Call.Name
		if rule.Call.Path != g.opts.ImportPath {
			call = rule.Call.Selector()
		}
		if rule.Param.Presence == vogue.ParamNone || use.Param == "" {
			return call + "(v)", nil
		}
		return call + "(v, " + quote(use.Param) + ")", nil
	}
	if rule.Emit == nil {
		return "", fmt.Errorf("gen: %s: rule %q has neither Emit nor Call", use.Pos, rule.Name)
	}
	return rule.Emit(emitContext(d, use)), nil
}

// emitContext builds the context handed to [vogue.Rule.Emit] and
// [vogue.Rule.Declare], so both always see exactly the same identifiers.
func emitContext(d parse.Directive, use parse.RuleUse) vogue.EmitContext {
	return vogue.EmitContext{Var: "v", Param: use.Param, Field: d.Field, Kind: d.Kind}
}

// renderMessage renders a rule message at generate time, rejecting a template
// that needs the runtime value.
func renderMessage(rule vogue.Rule, field, param string) (string, error) {
	message, err := rule.RenderMessage(vogue.MessageData{Field: field, Param: param, Value: valueSentinel})
	if err != nil {
		return "", err
	}
	if strings.Contains(message, valueSentinel) {
		return "", fmt.Errorf("rule %q: message template must not reference {{.Value}}: messages are rendered at generate time, and the offending value is already carried by vogue.FieldError.Value", rule.Name)
	}
	return message, nil
}

// docLines returns the godoc of a generated type: the comment the author wrote
// above the directive, or a sentence naming the field when they wrote none.
func docLines(d parse.Directive) []string {
	fallback := fmt.Sprintf("%s is the %s value object for the field %q.", d.Name, d.Kind, d.Field)
	if d.Doc == "" {
		return []string{fallback}
	}

	lines := strings.Split(d.Doc, "\n")
	if !strings.HasPrefix(lines[0], d.Name+" ") {
		lines = append([]string{fallback, ""}, lines...)
	}
	return lines
}

// receiver returns the one-letter method receiver of a type name.
func receiver(name string) string {
	return strings.ToLower(name[:1])
}
