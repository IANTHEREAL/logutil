package analyzer

import (
	"go/ast"
	"go/token"
	"go/types"
)

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
