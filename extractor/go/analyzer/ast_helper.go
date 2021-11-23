package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
)

// A visitFunc visits a node of the Go AST. The function can use stack to
// retrieve AST nodes on the path from the node up to the root.  If the return
// value is true, the children of node are also visited; otherwise they are
// skipped.
type visitFunc func(node ast.Node, stack stackFunc) bool

// A stackFunc returns the ith stack entry above of an AST node, where 0
// denotes the node itself. If the ith entry does not exist, the function
// returns nil.
type stackFunc func(i int) ast.Node

// astVisitor implements ast.Visitor, passing each visited node to a callback
// function.
type astVisitor struct {
	stack []ast.Node
	visit visitFunc
}

func newASTVisitor(f visitFunc) ast.Visitor { return &astVisitor{visit: f} }

// Visit implements the required method of the ast.Visitor interface.
func (w *astVisitor) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		w.stack = w.stack[:len(w.stack)-1] // pop
		return w
	}

	w.stack = append(w.stack, node) // push
	if !w.visit(node, w.parent) {
		return nil
	}
	return w
}

func (w *astVisitor) parent(i int) ast.Node {
	if i >= len(w.stack) {
		return nil
	}
	return w.stack[len(w.stack)-1-i]
}

// AstHelper takes package compilation data to help analyzer analyzes ast easily
type AstHelper struct {
	pkg       *types.Package
	typesInfo *types.Info
	token     *token.FileSet
}

func NewAstHelper(pkg *types.Package, fset *token.FileSet, typeinfo *types.Info) *AstHelper {
	return &AstHelper{
		pkg:       pkg,
		typesInfo: typeinfo,
		token:     fset,
	}
}

// types.Info holds result type information for a type-checked package.
// Only the information for which a map is provided is collected.
// If the package has type errors, the collected information may
// be incomplete.
func (helper *AstHelper) GetTypeInfo() *types.Info {
	return helper.typesInfo
}

// GetPos return the position of ast.node in format "file:line:column"
func (helper *AstHelper) GetPos(pos token.Pos) token.Position {
	return helper.token.Position(pos)
}

// GetPackage return type.Package of a package compilation
func (helper *AstHelper) GetPackage() *types.Package {
	return helper.pkg
}

// GetTypeUsed return the objects `id` denotes.
//
// For an embedded field, Uses returns the *TypeName it denotes.
//
// Invariant: Uses[id].Pos() != id.Pos()
func (helper *AstHelper) GetTypeUsed(id *ast.Ident) types.Object {
	return helper.typesInfo.Uses[id]
}

// GetTypeDef return the objects `id` define(including
// package names, dots "." of dot-imports, and blank "_" identifiers).
// For identifiers that do not denote objects (e.g., the package name
// in package clauses, or symbolic variables t in t := x.(type) of
// type switch headers), the corresponding objects are nil.
//
// For an embedded field, Defs returns the field *Var it defines.
//
// Invariant: Defs[id] == nil || Defs[id].Pos() == id.Pos()
func (helper *AstHelper) GetTypeDef(id *ast.Ident) types.Object {
	return helper.typesInfo.Defs[id]
}
