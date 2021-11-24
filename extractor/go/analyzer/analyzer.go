package analyzer

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"log"

	logpattern "github.com/IANTHEREAL/logutil/proto"
	"github.com/gogo/protobuf/proto"
)

// Analyzer used to analyze GO ast
type Aanalyzer interface {
	Run(*ast.File, *AstHelper)
	SetupOutput() <-chan proto.Message
	MarkDone()
}

// logAanalyzer used to find the log of interest
type logAanalyzer struct {
	// todo: add lock
	logChan chan proto.Message

	fn func(logPkg, logFn, logMessage string) (string, bool)
}

func NewAstAnalyzer(fn func(logPkg, logFn, logMessage string) (string, bool)) *logAanalyzer {
	return &logAanalyzer{fn: fn}
}

func (ai *logAanalyzer) Run(file *ast.File, helper *AstHelper) {
	ast.Walk(newASTVisitor(func(node ast.Node, stack stackFunc) bool {
		switch n := node.(type) {
		case *ast.FuncDecl:
			ai.visitFuncDecl(n, stack, helper)
		case *ast.FuncLit:
			ai.visitFuncLit(n, stack, helper)
		case *ast.BasicLit:
			ai.filter(n, stack, helper)
		}
		return true
	}), file)
}

func (ai *logAanalyzer) SetupOutput() <-chan proto.Message {
	ai.MarkDone()

	ai.logChan = make(chan proto.Message, 10000000)
	return ai.logChan
}

func (ai *logAanalyzer) MarkDone() {
	if ai.logChan != nil {
		close(ai.logChan)
	}
}

func (ai *logAanalyzer) filter(id *ast.BasicLit, stack stackFunc, helper *AstHelper) {
	switch p := stack(1).(type) {
	case *ast.Ident, *ast.SelectorExpr:
	case *ast.CallExpr:
		if pp, ok := p.Fun.(*ast.SelectorExpr); ok {
			ai.matchLog(id, pp.Sel, stack, helper)
		}
	}
}

func (ai *logAanalyzer) matchLog(l *ast.BasicLit, fn *ast.Ident, stack stackFunc, helper *AstHelper) {
	//log.Printf("match log %+v", l)
	obj := helper.GetTypeUsed(fn)
	if obj == nil {
		// Defining identifiers are handled by their parent nodes.
		return
	}

	if _, ok := isCall(fn, obj, stack); ok {
		callFnName, rawCallFnPos := ai.callContext(stack, helper)

		fnName := obj.Name()
		fnPkg := obj.Pkg().Name()
		helper.GetPos(obj.Pos())

		if fnName == "ErrorFilterContextCanceled" {
			//log.Fatalf("fnPkg %s", fnPkg)
		}

		if level, ok := ai.fn(fnPkg, fnName, l.Value); ok {
			fnPos := helper.GetPos(rawCallFnPos)
			fnProtoPos := &logpattern.Position{
				FilePath:     fnPos.Filename,
				LineNumber:   int32(fnPos.Line),
				ColumnOffset: int32(fnPos.Offset),
			}
			logPos := helper.GetPos(l.Pos())
			logProtoPos := &logpattern.Position{
				FilePath:     logPos.Filename,
				LineNumber:   int32(logPos.Line),
				ColumnOffset: int32(logPos.Offset),
			}

			fn := &logpattern.FuncInfo{
				Pos:  fnProtoPos,
				Name: callFnName,
			}

			ai.logChan <- &logpattern.LogPattern{
				Pos:       logProtoPos,
				Func:      fn,
				Level:     level,
				Signature: []string{l.Value},
			}
		}
	}
	//log.Printf("done match log %+v", l)
}

// visitFuncDecl handles function and method declarations and their parameters.
func (ai *logAanalyzer) visitFuncDecl(decl *ast.FuncDecl, stack stackFunc, helper *AstHelper) {
	// Get the type of this function, even if its name is blank.
	obj, _ := helper.GetTypeDef(decl.Name).(*types.Func)
	if obj == nil {
		return // a redefinition, for example
	}
}

// visitFuncLit handles function literals and their parameters.  The signature
// for a function literal is named relative to the signature of its parent
// function, or the file scope if the literal is at the top level.
func (ai *logAanalyzer) visitFuncLit(flit *ast.FuncLit, stack stackFunc, helper *AstHelper) {
	fi, _ := ai.callContext(stack, helper)
	if fi == "" {
		log.Fatalf("Function literal without a context: ", flit)
	}
}

// callContext returns funcInfo for the nearest enclosing parent function, not
// including the node itself, or the enclosing package initializer if the node
// is at the top level.
func (ai *logAanalyzer) callContext(stack stackFunc, helper *AstHelper) (string, token.Pos) {
	//log.Printf("call conext %s", stack(0))
	//defer log.Printf("done call conext %s", stack(0))
	for i := 1; ; i++ {
		switch p := stack(i).(type) {
		case *ast.FuncDecl:
			return p.Name.Name, p.Pos()
		case *ast.File:
			// Lazily emit a virtual node to represent the static
			// initializer for top-level expressions in this file of the
			// package.  We only do this if there are expressions that need
			// to be initialized.
			return fmt.Sprintf("<init>@%d", helper.GetPackage()), p.Pos()
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
