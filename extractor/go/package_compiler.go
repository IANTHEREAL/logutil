package extractor_go

import (
	"fmt"
	"go/build"
	"log"
	"strings"
)

// PackageCompiler helps to import package, and compile package to get AST set and type use info
// usage:
//  compiler := NewPackageComplier(build.Context)
//  pkg, err := compiler.Compile(importPath)   // get compiled package
//  or
//  pkg, err := compiler.ImportPackage(import) // to get compiled package
//  err = pkg.Compile()

type PackageCompiler struct {
	ctx            build.Context
	compliantCache map[string]*PackageCompilation // packageComplation.importPath -> packageComplation
}

func NewPackageComplier(ctx build.Context) *PackageCompiler {
	return &PackageCompiler{
		ctx:            ctx,
		compliantCache: make(map[string]*PackageCompilation),
	}
}

// Compile imports the package and compile it, return a compiled PackageCompilation
func (pc *PackageCompiler) Compile(importPath string) (*PackageCompilation, error) {
	// import all build packages under the import path
	pkg, err := pc.ImportPackage(importPath)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, fmt.Errorf("not found package(%s) to compile", importPath)
	}

	err = pkg.Compile()
	return pkg, err
}

// ImportPackage imports the package and all dependency packages, return a uncompiled PackageCompilation
func (pc *PackageCompiler) ImportPackage(importPath string) (*PackageCompilation, error) {
	listedPackages, err := pc.listPackages(pc.ctx, importPath)
	if err != nil {
		return nil, err
	}

	for _, pkg := range listedPackages {
		// ignore constructed test packages
		if pkg.ForTest != "" || strings.HasSuffix(pkg.ImportPath, ".test") {
			continue
		}

		if pkg.Error != nil {
			return nil, pkg.Error
		}

		pkgImportPatch := pkg.ImportPath
		_, ok := pc.compliantCache[pkgImportPatch]
		if !ok {
			pc.compliantCache[pkgImportPatch] = NewPackageCompilation(pkg.buildPackage(), pkg.DepOnly, pc.importDependPkg)
		}
	}

	return pc.compliantCache[importPath], nil
}

// importDependPkg helps package compilation to import dependency package, return the packageCompilation(depend-only)
func (pc *PackageCompiler) importDependPkg(importPath string, pkgBaseDir string) (*PackageCompilation, error) {
	// firstly find in cache
	if pkg := pc.compliantCache[importPath]; pkg != nil {
		return pkg.Clone(), nil
	}

	// then try to import
	bp, err := pc.ctx.Import("github.com/IANTHEREAL/logutil/pkg/util", pkgBaseDir, build.AllowBinary)
	if err != nil {
		log.Printf("import %v", err)
		return nil, err
	}

	pkg := NewPackageCompilation(bp, true, pc.importDependPkg)
	pc.compliantCache[bp.ImportPath] = pkg

	return pkg.Clone(), nil
}
