package log_extractor

import (
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/IANTHEREAL/logutil/extractor/go/compiler"
)

// Builer use to compile golang project into compilation package set
type Builder struct{}

func (b *Builder) Build(ctx build.Context, repoPath string) (*Repo, error) {
	repo := &Repo{
		repoRootPath: dirToImport(ctx, repoPath),
	}

	pkgPaths, err := fetchAllPkgs(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	compilations := make([]*compiler.PackageCompilation, 0, len(pkgPaths))
	c := compiler.NewPackageComplier(build.Default)
	for _, importPath := range pkgPaths {
		compliant, err := c.Compile(importPath)
		if err != nil {
			log.Printf("compile package %s failed, skip it", importPath)
			continue
		}

		compilations = append(compilations, compliant)
	}
	repo.pkgSet = compilations

	return repo, nil
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

type Repo struct {
	repoRootPath string
	pkgSet       []*compiler.PackageCompilation
}

func NewRepo(root string, pkgs []*compiler.PackageCompilation) *Repo {
	return &Repo{
		repoRootPath: root,
		pkgSet:       pkgs,
	}
}

func (r *Repo) ForEach(fn func(*compiler.PackageCompilation) error) error {
	for _, pkg := range r.pkgSet {
		if err := fn(pkg); err != nil {
			return err
		}
	}

	return nil
}

func (r *Repo) GetRepoPath() string {
	return r.repoRootPath
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
