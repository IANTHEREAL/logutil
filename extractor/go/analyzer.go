package extractor_go

import (
	"fmt"
	"go/ast"
	"go/types"
	"log"
)

type Aanalyzer interface {
	Run(*ast.File, *AstHelper)
}

type logAanalyzer struct {
	helper *AstHelper

	// todo: lock
	Functions    map[ast.Node]string
	PackageInits map[*ast.File]string
}

func NewAstAnalyzer() *logAanalyzer {
	return &logAanalyzer{
		Functions:    make(map[ast.Node]string),
		PackageInits: make(map[*ast.File]string),
	}
}

func (ai *logAanalyzer) Run(file *ast.File, helper *AstHelper) {
	ai.helper = helper
	ast.Walk(newASTVisitor(func(node ast.Node, stack stackFunc) bool {
		switch n := node.(type) {
		case *ast.Ident:
		//pcu.visitIdent(n, stack)
		case *ast.FuncDecl:
			ai.visitFuncDecl(n, stack)
		case *ast.FuncLit:
			ai.visitFuncLit(n, stack)
		case *ast.BasicLit:
			ai.isLog(n, stack)
		}
		return true
	}), file)
}

func (ai *logAanalyzer) isLog(id *ast.BasicLit, stack stackFunc) {
	//	log.Printf("stack %d, item %+v", i, stack(i))
	switch p := stack(1).(type) {
	case *ast.Ident, *ast.SelectorExpr:
		//log.Printf("stack %d, item %+v", i, p)
	case *ast.CallExpr:
		if pp, ok := p.Fun.(*ast.SelectorExpr); ok {
			ai.filterLog(id, pp.Sel, stack)
			//log.Printf("basiclist%+v stack %+v", id, p)
		}
	}
}

func (ai *logAanalyzer) filterLog(x *ast.BasicLit, id *ast.Ident, stack stackFunc) {
	obj := ai.helper.GetTypeUsed(id)
	if obj == nil {
		// Defining identifiers are handled by their parent nodes.
		return
	}

	if _, ok := isCall(id, obj, stack); ok {
		callName := ai.callContext(stack)
		fnName := obj.Name()
		fnPkg := obj.Pkg().Name()
		//log.Printf("^^^^^^^^^^^^^function %s.%s, is in %s", fnPkg, fnName, callName)

		if (fnPkg == "log") && (fnName == "Print" || fnName == "Printf") {
			//posInfo := fset.Position(id.Pos())
			log.Printf("***************%s log %s, belong to function %s", ai.helper.GetPos(id.Pos()), x.Value, callName)
		}
	}
}

// visitFuncDecl handles function and method declarations and their parameters.
func (ai *logAanalyzer) visitFuncDecl(decl *ast.FuncDecl, stack stackFunc) {
	// Get the type of this function, even if its name is blank.
	obj, _ := ai.helper.GetTypeDef(decl.Name).(*types.Func)
	if obj == nil {
		return // a redefinition, for example
	}
	ai.Functions[decl] = decl.Name.Name
}

// visitFuncLit handles function literals and their parameters.  The signature
// for a function literal is named relative to the signature of its parent
// function, or the file scope if the literal is at the top level.
func (ai *logAanalyzer) visitFuncLit(flit *ast.FuncLit, stack stackFunc) {
	fi := ai.callContext(stack)
	if fi == "" {
		log.Fatalf("Function literal without a context: ", flit)
	}
	ai.Functions[flit] = fi
}

// callContext returns funcInfo for the nearest enclosing parent function, not
// including the node itself, or the enclosing package initializer if the node
// is at the top level.
func (ai *logAanalyzer) callContext(stack stackFunc) string {
	for i := 1; ; i++ {
		switch p := stack(i).(type) {
		case *ast.FuncDecl:
			return p.Name.Name
		case *ast.FuncLit:
			return ai.Functions[p]
		case *ast.File:
			fi := ai.PackageInits[p]
			if fi == "" {
				// Lazily emit a virtual node to represent the static
				// initializer for top-level expressions in this file of the
				// package.  We only do this if there are expressions that need
				// to be initialized.
				fi = fmt.Sprintf("<init>@%d", ai.helper.GetPackage())
				ai.PackageInits[p] = fi
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
