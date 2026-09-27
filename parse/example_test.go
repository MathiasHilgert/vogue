package parse_test

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"

	"github.com/MathiasHilgert/vogue"
	"github.com/MathiasHilgert/vogue/parse"
)

// exampleRules is the catalogue the examples validate directives against.
func exampleRules() *vogue.RuleSet {
	always := func(vogue.EmitContext) string { return "true" }
	set := &vogue.RuleSet{}
	_ = set.Add(
		vogue.Rule{
			Name: "required", Kinds: vogue.Kinds(vogue.String),
			Message: "{{.Field}} is required", Emit: always,
		},
		vogue.Rule{
			Name: "min", Kinds: vogue.Kinds(vogue.String, vogue.Int),
			Param:   vogue.ParamSpec{Presence: vogue.ParamRequired, Type: vogue.ParamInt},
			Message: "{{.Field}} must be at least {{.Param}}", Emit: always,
		},
	)
	return set
}

func ExampleLine() {
	kind, name, tokens, err := parse.Line(`//vogue:string Title required min=1`)
	fmt.Println(kind, name, tokens, err)
	// Output: string Title [required min=1] <nil>
}

func ExampleFieldName() {
	fmt.Println(parse.FieldName("Title"))
	fmt.Println(parse.FieldName("TabID"))
	fmt.Println(parse.FieldName("CUITNumber"))
	// Output:
	// title
	// tabID
	// cuitNumber
}

func ExampleFiles() {
	src := `package tab

// Title is the name of a tab.
//vogue:string Title required min=1
//vogue:enum TabStatus open,closed
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "vo.go", src, parser.ParseComments)
	if err != nil {
		panic(err)
	}

	pkg, err := parse.Files(fset, []*ast.File{file}, exampleRules())
	if err != nil {
		panic(err)
	}
	for _, d := range pkg.Directives() {
		fmt.Printf("%s %s field=%s rules=%d values=%d\n", d.Kind, d.Name, d.Field, len(d.Rules), len(d.Values))
	}
	// Output:
	// string Title field=title rules=2 values=0
	// enum TabStatus field=tabStatus rules=0 values=2
}

func ExampleErrors() {
	src := "package tab\n//vogue:string Title mni=1\n//vogue:int Covers min\n"

	fset := token.NewFileSet()
	file, _ := parser.ParseFile(fset, "vo.go", src, parser.ParseComments)

	_, err := parse.Files(fset, []*ast.File{file}, exampleRules())
	fmt.Println(err)
	// Output:
	// vo.go:2:22: unknown rule "mni" for kind string (did you mean "min"?)
	// vo.go:3:20: rule "min" requires a parameter (write min=<int>)
}
