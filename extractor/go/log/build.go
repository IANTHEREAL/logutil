package log_extractor

import (
	"go/build"
	"os"
	"path/filepath"
	"strings"

	"github.com/IANTHEREAL/logutil/extractor/go/compiler"
)

func Build(ctx build.Context, repoPath string) ([]*compiler.PackageCompilation, error) {
	pkgPaths, err := fetchAllPkgs(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	compilations := make([]*compiler.PackageCompilation, 0, len(pkgPaths))
	c := compiler.NewPackageComplier(build.Default)
	for _, importPath := range pkgPaths {
		compliant, err := c.Compile(importPath)
		if err != nil {
			return compilations, err
		}

		compilations = append(compilations, compliant)
	}

	return compilations, nil

}

func fetchAllPkgs(ctx build.Context, repoPath string) ([]string, error) {
	dirMap := make(map[string]struct{})
	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if filepath.Ext(path) == ".go" {
				dirMap[filepath.Dir(path)] = struct{}{}
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	pkgDirs := make([]string, 0, len(dirMap))
	for pkg := range dirMap {
		pkgDirs = append(pkgDirs, dirToImport(ctx, pkg))
	}

	return pkgDirs, nil
}

func dirToImport(ctx build.Context, dir string) string {
	for _, src := range ctx.SrcDirs() {
		if strings.HasPrefix(dir, src) {
			if rel, err := filepath.Rel(src, dir); err == nil {
				return rel
			}
		}
	}
	return dir
}
