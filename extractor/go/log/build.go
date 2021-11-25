package log_extractor

import (
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/IANTHEREAL/logutil/extractor/go/compiler"
)

// Builer used to compile golang project into compilation package set
// input - ctx is go/build Conext with customize the variables for go build
// input - repoPath is the directory path where project under $GOPATH or $GOPATH
// return - Repo is a object contains package compilation set
/* uasage:
    builder := &logextractor.Builder{}
	...
	path, err := filepath.Abs("./")
	or
	path, err := filepath.Abs("/Users/ianz/Work/go/src/github.com/pingcap/ticdc/dm")
	...
	repo, err := builder.Build(build.Default, path)
*/
type Builder struct{}

func (b *Builder) Build(ctx build.Context, repoPath string) (*Repo, error) {
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

	return NewRepo(dirToImport(ctx, repoPath), compilations), nil
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

// Repo is a object contains package compilation set
// it provide ForEach() function to let caller visit every package compilation serially
/* uasage:
    repo := NewRepo(repo)
	...
	repo.ForEach(func(p *compiler.PackageCompilation) error {
		log.Printf("package compilation %+v", p)
		...
	})
*/
type Repo struct {
	repoRoot string
	pkgSet   []*compiler.PackageCompilation
}

func NewRepo(root string, pkgs []*compiler.PackageCompilation) *Repo {
	return &Repo{
		repoRoot: root,
		pkgSet:   pkgs,
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
	return r.repoRoot
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
