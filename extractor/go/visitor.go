package extractor_go

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
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

func (pcu *PackageComplation) isLog(id *ast.BasicLit, stack stackFunc) {
	//	log.Printf("stack %d, item %+v", i, stack(i))
	switch p := stack(1).(type) {
	case *ast.Ident, *ast.SelectorExpr:
		//log.Printf("stack %d, item %+v", i, p)
	case *ast.CallExpr:
		if pp, ok := p.Fun.(*ast.SelectorExpr); ok {
			pcu.filterLog(id, pp.Sel, stack)
			//log.Printf("basiclist%+v stack %+v", id, p)
		}
	}
}

func (pcu *PackageComplation) filterLog(x *ast.BasicLit, id *ast.Ident, stack stackFunc) {
	obj := pcu.TypesInfo.Uses[id]
	if obj == nil {
		// Defining identifiers are handled by their parent nodes.
		return
	}

	if _, ok := isCall(id, obj, stack); ok {
		callName := pcu.callContext(stack)
		fnName := obj.Name()
		fnPkg := obj.Pkg().Name()
		//log.Printf("^^^^^^^^^^^^^function %s.%s, is in %s", fnPkg, fnName, callName)

		if (fnPkg == "log") && (fnName == "Print" || fnName == "Printf") {
			posInfo := pcu.FileSet.Position(id.Pos())
			log.Printf("***************%s log %s, belong to function %s", posInfo, x.Value, callName)
		}
	}
}

// visitFuncDecl handles function and method declarations and their parameters.
func (pcu *PackageComplation) visitFuncDecl(decl *ast.FuncDecl, stack stackFunc) {
	// Get the type of this function, even if its name is blank.
	obj, _ := pcu.TypesInfo.Defs[decl.Name].(*types.Func)
	if obj == nil {
		return // a redefinition, for example
	}
	pcu.functions[decl] = decl.Name.Name
}

// visitFuncLit handles function literals and their parameters.  The signature
// for a function literal is named relative to the signature of its parent
// function, or the file scope if the literal is at the top level.
func (pcu *PackageComplation) visitFuncLit(flit *ast.FuncLit, stack stackFunc) {
	fi := pcu.callContext(stack)
	if fi == "" {
		log.Fatalf("Function literal without a context: ", flit)
	}
	pcu.functions[flit] = fi
}

// callContext returns funcInfo for the nearest enclosing parent function, not
// including the node itself, or the enclosing package initializer if the node
// is at the top level.
func (pcu *PackageComplation) callContext(stack stackFunc) string {
	for i := 1; ; i++ {
		switch p := stack(i).(type) {
		case *ast.FuncDecl, *ast.FuncLit:
			return pcu.functions[p]
		case *ast.File:
			fi := pcu.packageInits[p]
			if fi == "" {
				// Lazily emit a virtual node to represent the static
				// initializer for top-level expressions in this file of the
				// package.  We only do this if there are expressions that need
				// to be initialized.
				fi = fmt.Sprintf("<init>@%d", p.Package)
				pcu.packageInits[p] = fi
			}
			return fi
		}
	}
}

func isCall(id *ast.Ident, obj types.Object, stack stackFunc) (*ast.CallExpr, bool) {
	if _, ok := obj.(*types.Func); ok {
		if cal, ok := stack(1).(*ast.CallExpr); ok {
			if sel, ok := cal.Fun.(*ast.SelectorExpr); ok && sel.Sel == id {
				return cal, true // x.id(...)
			}
		}
	}
	return nil, false
}
