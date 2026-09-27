package gen

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
)

// negate returns the Go expression that is true exactly when expr is false.
//
// A rule emits the condition under which a value is valid, and the
// constructor records a failure when it does not hold. Writing that as
// `if !(expr)` is what staticcheck's QF1001 flags as a missed De Morgan, so the
// negation is pushed into the expression instead: a comparison flips its
// operator, a conjunction becomes a disjunction of negations and the other way
// round, a double negation cancels, and anything else — a call, an identifier
// — is prefixed with `!`.
func negate(expr string) (string, error) {
	parsed, err := parser.ParseExpr(expr)
	if err != nil {
		return "", fmt.Errorf("gen: rule expression %q does not parse: %w", expr, err)
	}
	var b bytes.Buffer
	if err := printer.Fprint(&b, token.NewFileSet(), not(parsed)); err != nil {
		return "", fmt.Errorf("gen: printing the negation of %q: %w", expr, err)
	}
	return b.String(), nil
}

// inverse maps a comparison operator to the one that holds exactly when it
// does not.
var inverse = map[token.Token]token.Token{
	token.EQL: token.NEQ,
	token.NEQ: token.EQL,
	token.LSS: token.GEQ,
	token.GEQ: token.LSS,
	token.GTR: token.LEQ,
	token.LEQ: token.GTR,
}

// not builds the negation of one expression.
func not(e ast.Expr) ast.Expr {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return not(e.X)
	case *ast.UnaryExpr:
		if e.Op == token.NOT {
			return unparen(e.X)
		}
	case *ast.BinaryExpr:
		if op, ok := inverse[e.Op]; ok {
			return &ast.BinaryExpr{X: e.X, Op: op, Y: e.Y}
		}
		switch e.Op {
		case token.LAND:
			return &ast.BinaryExpr{X: grouped(not(e.X), token.LOR), Op: token.LOR, Y: grouped(not(e.Y), token.LOR)}
		case token.LOR:
			return &ast.BinaryExpr{X: grouped(not(e.X), token.LAND), Op: token.LAND, Y: grouped(not(e.Y), token.LAND)}
		default:
		}
		return &ast.UnaryExpr{Op: token.NOT, X: &ast.ParenExpr{X: e}}
	}
	return &ast.UnaryExpr{Op: token.NOT, X: e}
}

// grouped parenthesises e when it is a binary expression binding more loosely
// than the operator it becomes an operand of, which go/printer would not do on
// its own.
func grouped(e ast.Expr, parent token.Token) ast.Expr {
	if b, ok := e.(*ast.BinaryExpr); ok && b.Op.Precedence() < parent.Precedence() {
		return &ast.ParenExpr{X: e}
	}
	return e
}

// unparen strips the parentheses around e.
func unparen(e ast.Expr) ast.Expr {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			return e
		}
		e = p.X
	}
}
