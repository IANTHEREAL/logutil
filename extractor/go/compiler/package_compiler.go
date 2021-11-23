package compiler

import (
	"fmt"
	"go/build"
	"strings"
)

// PackageCompiler helps to compile package, the analysis algorithm can be run on the compiled package
// The compiler only to be initialized once, and then used multiple times
// usage:
//  compiler := NewPackageComplier(build.Context)
//  ....
//  pkg, err := compiler.Compile(importPath)   // get compiled package
//  ....
//  pkg.RunAnalyze(Analyzer)
//  it is not concurrency safe
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

// Compile compile the package, return a compiled PackageCompilation that can run analysis
func (pc *PackageCompiler) Compile(importPath string) (*PackageCompilation, error) {
	// import the package and dependency packages under the import path
	pkg, err := pc.importPackage(importPath)
	if err != nil {
		return nil, err
	}
	if pkg == nil {
		return nil, fmt.Errorf("not found package(%s) to compile", importPath)
	}
	// compile package
	err = pkg.Compile()
	return pkg, err
}

// importPackage imports the package and dependency packages under the import path, return a uncompiled PackageCompilation
func (pc *PackageCompiler) importPackage(importPath string) (*PackageCompilation, error) {
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

// importDependPkg helps to import dependency package, return a packageCompilation(depend-only)
func (pc *PackageCompiler) importDependPkg(importPath string, pkgBaseDir string) (*PackageCompilation, error) {
	// firstly find in cache
	if pkg := pc.compliantCache[importPath]; pkg != nil {
		return pkg.Clone(), nil
	}

	// then try to import package using golang/build.Import
	bp, err := pc.ctx.Import(importPath, pkgBaseDir, build.AllowBinary)
	if err != nil {
		return nil, err
	}

	pkg := NewPackageCompilation(bp, true, pc.importDependPkg)
	pc.compliantCache[bp.ImportPath] = pkg

	return pkg.Clone(), nil
}
