package extractor_go

import (
	"go/ast"
	"go/build"
	"go/token"
	"go/types"

	proto_common "github.com/IANTHEREAL/logutil/pkg/proto/common_go"
)

var (
	bc = build.Default
)

type PackageInfo struct {
	Name         proto_common.XVName
	ImportPath   string
	Package      *types.Package
	Dependencies map[string]*types.Package
	FileSet      *token.FileSet
	Files        []*ast.File
}

func (pi *PackageInfo) Build() error {
	return nil
}
