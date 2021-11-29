package log_extractor

import (
	"errors"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/IANTHEREAL/logutil/extractor/go/compiler"
)

// Builer used to compile golang project into multiple package compilations
// input - ctx is go/build Conext with customize the variables for build golang package
// input - repoPath is the directory path where project under $GOPATH or $GOROOT
// return - Repo is a object contains all package compilations under the repo
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
	importPath, err := dirToImport(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	pkgPaths, err := fetchAllPkgs(ctx, repoPath)
	if err != nil {
		return nil, err
	}

	compilations := make([]*compiler.PackageCompilation, 0, len(pkgPaths))
	c := compiler.NewPackageComplier(build.Default)

	// compile packages concurrently
	wg := sync.WaitGroup{}
	ch := make(chan *compiler.PackageCompilation, len(pkgPaths))
	startTime := time.Now()
	for _, importPath := range pkgPaths {
		wg.Add(1)
		go func(importPath string) {
			compilation, err := c.Compile(importPath)
			if err != nil {
				log.Printf("compile package %s failed: %v, skip it", importPath, err)
			} else {
				ch <- compilation
			}
			wg.Done()
		}(importPath)
	}
	wg.Wait()
	close(ch)
	log.Printf("compile package cost time %s", time.Since(startTime))

	for {
		compilation, ok := <-ch
		if !ok {
			break
		}
		compilations = append(compilations, compilation)
	}
	return NewRepo(importPath, compilations), nil
}

// fetchAllPkgs finds all directories that contains at least one go source file,
// one directory is one package
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
		// try to compute import path
		pkgImportPath, _ := dirToImport(ctx, pkg)
		pkgDirs = append(pkgDirs, pkgImportPath)
	}

	return pkgDirs, nil
}

// Repo is a object contains multiple package compilations
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

var ErrNotSupportLocalImport = errors.New("not support local import, please put repo under right path of the $GOPATH, e.g. $GOPATH/src/github.com/org/repo")

// try to compute import path relative to GOPATH or GOROOT.
// Now we only support import package that under GOPATH/GOROOT, otherwise return ErrNotSupportLocalImport
func dirToImport(ctx build.Context, dir string) (string, error) {
	for _, src := range ctx.SrcDirs() {
		if strings.HasPrefix(dir, src) {
			if rel, err := filepath.Rel(src, dir); err == nil {
				return rel, nil
			}
		}
	}
	return dir, ErrNotSupportLocalImport
}
