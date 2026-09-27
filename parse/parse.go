// Package parse reads vogue directives out of Go source files and turns them
// into a validated model the generator can consume.
//
// A directive is a line comment naming a kind, a type name and, for the string
// and int kinds, the rules to apply in evaluation order:
//
//	//vogue:string Title     required trim min=1 max=120
//	//vogue:enum   TabStatus open,closed,voided
//	//vogue:id     TabID
//
// Every problem found in a package is reported, not just the first, and each
// diagnostic carries the position of the offending token so the output reads
// like a compiler error.
package parse

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/govalues/decimal"

	"github.com/MathiasHilgert/vogue"
)

var (
	// typeNameRe is the shape of a value-object type name: it must be an
	// exported Go identifier so the generated type is usable from other
	// packages.
	typeNameRe = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	// enumValueRe is the shape of an enum wire value.
	enumValueRe = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	// generatedRe matches the header that marks a file as generated, so
	// re-running the generator never reads its own output.
	generatedRe = regexp.MustCompile(`(?m)^// Code generated .* DO NOT EDIT\.$`)
)

// Dir parses every hand-written Go file of one directory. Test files and files
// carrying the generated-code header are skipped, which makes running the
// generator over its own output idempotent.
func Dir(dir string, rules *vogue.RuleSet) (*Package, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	fset := token.NewFileSet()
	var files []*ast.File
	for _, entry := range entries {
		if entry.IsDir() || !isSourceName(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		src, err := os.ReadFile(path) //nolint:gosec // the caller chose the directory to generate from.
		if err != nil {
			return nil, err
		}
		if isGenerated(src) {
			continue
		}
		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	pkg, err := Files(fset, files, rules)
	if err != nil {
		return nil, err
	}
	pkg.Path = dir
	return pkg, nil
}

// isSourceName reports whether the file is a hand-written Go source file.
func isSourceName(name string) bool {
	return strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// isGenerated reports whether the source carries the generated-code header
// before its package clause.
func isGenerated(src []byte) bool {
	head := src
	if i := strings.Index(string(src), "\npackage "); i >= 0 {
		head = src[:i]
	}
	return generatedRe.Match(head)
}

// Files parses already-loaded syntax trees. It is the form to use when the
// sources come from somewhere other than a directory, such as a test or an
// embedding tool.
func Files(fset *token.FileSet, files []*ast.File, rules *vogue.RuleSet) (*Package, error) {
	if rules == nil {
		rules = &vogue.RuleSet{}
	}

	p := &collector{fset: fset, rules: rules, declared: map[string]token.Position{}}
	pkg := &Package{}
	for _, file := range files {
		parsed := p.file(file)
		if pkg.Name == "" {
			pkg.Name = file.Name.Name
		} else if file.Name.Name != pkg.Name {
			p.errs.err(fset.Position(file.Name.Pos()),
				fmt.Sprintf("found packages %q and %q in the same directory", pkg.Name, file.Name.Name))
		}
		pkg.Files = append(pkg.Files, parsed)
	}

	p.checkCatalogues(pkg)
	if len(p.errs) > 0 {
		p.errs.sort()
		return nil, p.errs
	}
	return pkg, nil
}

// collector carries the state shared by every directive of one package: the rule
// catalogue, the names already declared and the diagnostics collected so far.
type collector struct {
	fset     *token.FileSet
	rules    *vogue.RuleSet
	declared map[string]token.Position
	errs     Errors
}

// catalogueMethods are the methods every enum catalogue declares, which no
// member method may shadow.
var catalogueMethods = map[string]struct{}{"All": {}, "Parse": {}}

// checkCatalogues reports an enum whose catalogue type would take the name of
// another value object of the package. It runs once every directive is known,
// because the colliding one may be declared later or in another file.
func (p *collector) checkCatalogues(pkg *Package) {
	for _, d := range pkg.Directives() {
		if d.Catalogue == "" {
			continue
		}
		if at, taken := p.declared[d.Catalogue]; taken {
			p.errs.hint(d.Pos,
				fmt.Sprintf("the catalogue type %s of enum %q collides with a value object declared at %s", d.Catalogue, d.Name, at),
				"rename one of them")
		}
	}
}

// file collects the directives of one syntax tree, remembering the ordinary
// comment lines written directly above each of them as its doc.
func (p *collector) file(file *ast.File) File {
	out := File{Path: p.fset.Position(file.Package).Filename}
	for _, group := range file.Comments {
		var doc []string
		for _, comment := range group.List {
			text := comment.Text
			if !strings.HasPrefix(text, prefix) {
				if strings.HasPrefix(text, "//") {
					doc = append(doc, strings.TrimPrefix(strings.TrimPrefix(text, "//"), " "))
				} else {
					doc = nil
				}
				continue
			}
			if d, ok := p.directive(comment, strings.Join(doc, "\n")); ok {
				out.Directives = append(out.Directives, d)
			}
			doc = nil
		}
	}
	return out
}

// directive validates one directive comment and turns it into a model entry.
// It returns false when the directive is broken, having recorded why.
func (p *collector) directive(comment *ast.Comment, doc string) (Directive, bool) {
	pos := p.fset.Position(comment.Slash)

	kind, nameTok, tokens, lerr := parseLine(comment.Text)
	if lerr != nil {
		lerr.Pos = pos
		p.errs = append(p.errs, lerr)
		return Directive{}, false
	}

	namePos := offsetPos(pos, nameTok.off)
	if !typeNameRe.MatchString(nameTok.text) {
		p.errs.hint(namePos, fmt.Sprintf("invalid type name %q", nameTok.text), "want "+typeNameRe.String())
		return Directive{}, false
	}
	if first, ok := p.declared[nameTok.text]; ok {
		p.errs.hint(pos, fmt.Sprintf("duplicate value object %q", nameTok.text), "first declared at "+first.String())
		return Directive{}, false
	}
	p.declared[nameTok.text] = pos

	d := Directive{
		Pos:   pos,
		Kind:  kind,
		Name:  nameTok.text,
		Field: FieldName(nameTok.text),
		Doc:   doc,
	}

	switch kind {
	case vogue.ID:
		strategy, ok := p.idStrategy(pos, tokens)
		if !ok {
			return Directive{}, false
		}
		d.Strategy = strategy
	case vogue.Enum:
		values, ok := p.enumValues(pos, namePos, nameTok.text, tokens)
		if !ok {
			return Directive{}, false
		}
		d.Values = values
		d.Catalogue = plural(nameTok.text)
	case vogue.String, vogue.Int, vogue.Decimal:
		rules, examples, ok := p.ruleUses(pos, kind, tokens)
		if !ok {
			return Directive{}, false
		}
		d.Rules, d.Examples = rules, examples
	}
	return d, true
}

// idStrategy resolves the optional strategy token of an id directive,
// defaulting to [IDUUIDv7] when it is absent.
func (p *collector) idStrategy(pos token.Position, tokens []dtoken) (IDStrategy, bool) {
	if len(tokens) == 0 {
		return IDUUIDv7, true
	}
	if len(tokens) > 1 {
		p.errs.err(offsetPos(pos, tokens[1].off),
			fmt.Sprintf("kind id takes at most one strategy, got %q", tokens[1].text))
		return 0, false
	}

	tok := tokens[0]
	strategy, ok := parseIDStrategy(tok.text)
	if ok {
		return strategy, true
	}

	names := idStrategyNames[:]
	e := &Error{Pos: offsetPos(pos, tok.off), Msg: fmt.Sprintf("unknown id strategy %q", tok.text)}
	if near, hasNear := nearest(tok.text, names); hasNear {
		e.Hint = fmt.Sprintf("did you mean %q?", near)
	} else {
		e.Hint = "want one of: " + strings.Join(names, ", ")
	}
	p.errs = append(p.errs, e)
	return 0, false
}

// enumValues validates the single comma-separated token of an enum directive
// and derives the Go constant name of every member.
func (p *collector) enumValues(pos, namePos token.Position, name string, tokens []dtoken) ([]EnumValue, bool) {
	switch {
	case len(tokens) == 0:
		p.errs.err(namePos, fmt.Sprintf("enum %q requires a comma-separated list of values", name))
		return nil, false
	case len(tokens) > 1:
		p.errs.err(offsetPos(pos, tokens[1].off),
			fmt.Sprintf("enum %q takes exactly one comma-separated list of values", name))
		return nil, false
	}

	tok := tokens[0]
	tokPos := offsetPos(pos, tok.off)
	items := strings.Split(tok.text, ",")
	if len(items) < 2 {
		p.errs.err(tokPos, fmt.Sprintf("enum %q must declare at least 2 values, got %d", name, len(items)))
		return nil, false
	}

	seen := make(map[string]struct{}, len(items))
	values := make([]EnumValue, 0, len(items))
	ok := true
	for _, item := range items {
		if !enumValueRe.MatchString(item) {
			p.errs.hint(tokPos, fmt.Sprintf("invalid enum value %q", item), "want "+enumValueRe.String())
			ok = false
			continue
		}
		if _, dup := seen[item]; dup {
			p.errs.err(tokPos, fmt.Sprintf("duplicate enum value %q", item))
			ok = false
			continue
		}
		seen[item] = struct{}{}
		method := methodName(item)
		if _, reserved := catalogueMethods[method]; reserved {
			p.errs.hint(tokPos,
				fmt.Sprintf("enum member %q would be generated as the method %s, which the catalogue already declares", item, method),
				"rename the member")
			ok = false
			continue
		}
		values = append(values, EnumValue{Value: item, Const: constName(name, item), Method: method})
	}
	return values, ok
}

// exampleToken is the reserved token name that declares a valid value rather
// than naming a rule.
const exampleToken = "example"

// ruleUses resolves every rule token of a string, int or decimal directive
// against the catalogue, preserving the order in which they were written, and
// collects its `example=` tokens.
func (p *collector) ruleUses(pos token.Position, kind vogue.Kind, tokens []dtoken) ([]RuleUse, []string, bool) {
	uses := make([]RuleUse, 0, len(tokens))
	var examples []string
	seen := make(map[string]token.Position, len(tokens))
	ok := true
	for _, tok := range tokens {
		tokPos := offsetPos(pos, tok.off)
		name, param, hasParam := strings.Cut(tok.text, "=")

		if name == exampleToken {
			if !p.checkExample(tokPos, kind, param, hasParam) {
				ok = false
				continue
			}
			examples = append(examples, param)
			continue
		}

		rule, found := p.rules.Get(name)
		if !found {
			e := &Error{Pos: tokPos, Msg: fmt.Sprintf("unknown rule %q for kind %s", name, kind)}
			if near, hasNear := p.rules.Suggest(name); hasNear {
				e.Hint = fmt.Sprintf("did you mean %q?", near)
			}
			p.errs = append(p.errs, e)
			ok = false
			continue
		}
		if !rule.Kinds.Has(kind) {
			p.errs.hint(tokPos, fmt.Sprintf("rule %q does not apply to kind %s", name, kind),
				"it applies to: "+rule.Kinds.String())
			ok = false
			continue
		}
		if first, dup := seen[name]; dup {
			p.errs.hint(tokPos, fmt.Sprintf("duplicate rule %q", name), "already given at "+first.String())
			ok = false
			continue
		}
		seen[name] = tokPos

		if !p.checkParam(tokPos, kind, rule, param, hasParam) {
			ok = false
			continue
		}
		uses = append(uses, RuleUse{Rule: rule, Param: param, Pos: tokPos})
	}
	return uses, examples, ok
}

// checkExample validates an `example=` token: it needs a value, and the value
// must be one the kind's constructor can be handed.
func (p *collector) checkExample(pos token.Position, kind vogue.Kind, value string, hasValue bool) bool {
	if !hasValue {
		p.errs.hint(pos, "example requires a value", "write example=<value>")
		return false
	}
	var err error
	switch kind {
	case vogue.Int:
		_, err = strconv.ParseInt(value, 10, 64)
	case vogue.Decimal:
		_, err = decimal.Parse(value)
	default:
		// A string takes any text, and the enum and id kinds take no tokens.
	}
	if err != nil {
		p.errs.err(pos, fmt.Sprintf("example %q is not a valid %s", value, kind))
		return false
	}
	return true
}

// checkParam validates the presence and the syntax of a rule parameter against
// the rule's declared contract.
func (p *collector) checkParam(pos token.Position, kind vogue.Kind, rule vogue.Rule, param string, hasParam bool) bool {
	switch rule.Param.Presence {
	case vogue.ParamNone:
		if hasParam {
			p.errs.hint(pos, fmt.Sprintf("rule %q takes no parameter", rule.Name), "write "+rule.Name)
			return false
		}
		return true
	case vogue.ParamRequired:
		if !hasParam {
			p.errs.hint(pos, fmt.Sprintf("rule %q requires a parameter", rule.Name),
				fmt.Sprintf("write %s=<%s>", rule.Name, rule.Param.Type))
			return false
		}
	case vogue.ParamOptional:
		if !hasParam {
			return true
		}
	}

	if msg, bad := badParam(kind, rule, param); bad {
		p.errs.err(pos, fmt.Sprintf("invalid parameter for rule %q: %s", rule.Name, msg))
		return false
	}
	return true
}

// badParam reports why the parameter does not satisfy the rule's parameter
// type, if it does not.
//
// The kind is part of the question because [vogue.ParamNumber] means a
// different number on each kind: an int64 wherever the generated comparison is
// an int64 one — the rune count of a string, the value of an integer — and an
// exact decimal on the decimal kind. That is what keeps `min=0.5` a
// generate-time error on an int while accepting it on a rate.
func badParam(kind vogue.Kind, rule vogue.Rule, param string) (string, bool) {
	switch rule.Param.Type {
	case vogue.ParamNumber:
		if kind == vogue.Decimal {
			return badDecimalParam(param)
		}
		fallthrough
	case vogue.ParamInt:
		if _, err := strconv.ParseInt(param, 10, 64); err != nil {
			return fmt.Sprintf("%q is not an integer", param), true
		}
	case vogue.ParamList:
		for i, item := range strings.Split(param, ",") {
			if item == "" {
				return fmt.Sprintf("item %d of %q is empty", i+1, param), true
			}
			if kind != vogue.Int {
				continue
			}
			if _, err := strconv.ParseInt(item, 10, 64); err != nil {
				return fmt.Sprintf("item %d of %q is not an integer", i+1, param), true
			}
		}
	case vogue.ParamRegex:
		if _, err := regexp.Compile(param); err != nil {
			return err.Error(), true
		}
	case vogue.ParamString:
	}
	return "", false
}

// badDecimalParam reports why a decimal parameter cannot be used as written.
//
// [decimal.Parse] rounds a literal carrying more than 19 decimal places
// instead of refusing it, which would quietly turn a bound into a slightly
// different bound. A generator that silently moves a boundary is worse than
// one that complains, so the rounding is detected here and reported. A literal
// written in exponent notation is exempt, because the digits after its point
// are not its decimal places.
func badDecimalParam(param string) (string, bool) {
	parsed, err := decimal.Parse(param)
	if err != nil {
		return fmt.Sprintf("%q is not a decimal number: %v", param, err), true
	}
	if strings.ContainsAny(param, "eE") {
		return "", false
	}
	if _, frac, ok := strings.Cut(param, "."); ok && len(frac) > parsed.Scale() {
		return fmt.Sprintf("%q has more than %d decimal places, which is the most a decimal can hold; "+
			"write the bound you mean", param, decimal.MaxScale), true
	}
	return "", false
}

// offsetPos moves a position forward by a byte offset inside the same line,
// which is how a directive position becomes a token position.
func offsetPos(pos token.Position, off int) token.Position {
	pos.Offset += off
	pos.Column += off
	return pos
}
