package extractor_go

import (
	"fmt"
	"go/build"
	"log"
	"strings"
)

type PackageComplier struct {
	ctx        build.Context
	importPath string
	compliants map[string]*PackageComplation // packageComplation.importPath -> packageComplation
}

func NewPackageComplier(ctx build.Context, importPath string) *PackageComplier {
	return &PackageComplier{
		ctx:        ctx,
		importPath: importPath,
		compliants: make(map[string]*PackageComplation),
	}
}

func (pc *PackageComplier) Compile() (*PackageComplation, error) {
	// import all build packages under the import path
	if err := pc.Import(); err != nil {
		return nil, err
	}

	// exclude depend only pkg
	var compliant *PackageComplation
	for _, unit := range pc.compliants {
		if unit.DepOnly {
			continue
		} else if compliant != nil {
			return nil, fmt.Errorf("compling package: multiple packages %s and %s", unit.ImportPath, compliant.ImportPath)
		}
		compliant = unit
	}

	// resolve package compliation detail
	err := compliant.Resolve()
	return compliant, err
}

func (pc *PackageComplier) Import() error {
	listedPackages, err := pc.listPackages(pc.ctx, pc.importPath)
	if err != nil {
		return err
	}

	for _, pkg := range listedPackages {
		if pkg.ForTest != "" || strings.HasSuffix(pkg.ImportPath, ".test") {
			// ignore constructed test packages
			continue
		} else if pkg.Error != nil {
			return pkg.Error
		}

		importPath := pkg.ImportPath
		_, ok := pc.compliants[importPath]
		if !ok {
			pc.compliants[pkg.ImportPath] = NewPackageComplation(pkg.buildPackage(), pkg.DepOnly, pc.findDependComplation)
		}
	}

	return nil
}

func (pc *PackageComplier) findDependComplation(importPath string, pkgBaseDir string) (*PackageComplation, error) {
	if unit := pc.compliants[importPath]; unit != nil {
		return unit.Clone(), nil
	}

	bp, err := pc.ctx.Import(importPath, pkgBaseDir, build.AllowBinary)
	if err != nil {
		log.Printf("import %v", err)
		return nil, err
	}

	unit := NewPackageComplation(bp, true, pc.findDependComplation)
	pc.compliants[bp.ImportPath] = unit

	return unit.Clone(), nil
}
