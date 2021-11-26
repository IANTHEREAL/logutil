package compiler

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"

	"github.com/IANTHEREAL/logutil/extractor/go/analyzer"
	"github.com/IANTHEREAL/logutil/pkg/util"
	logpattern "github.com/IANTHEREAL/logutil/proto"
	"golang.org/x/tools/go/gcexportdata"
)

// PackageCompilation helps to compile package to get AST set and type use info
// usage:
//  compilation := NewPackageCompilation(build.Package, depOnly, importDependFn)
//  err := compilation.Compile()   // compile package to get file AST
//  ...
//  fileCompilation := compilation.ForEach(fn func(*FileCompilation, *analysis.AstHelper)) // do analysis on file compliation
//  it is not concurrency safe
type PackageCompilation struct {
	// read only
	ImportPath    string
	PackagePath   *logpattern.PackagePath
	BuiledPackage *build.Package
	DepOnly       bool

	// written at compiling
	Deps          map[string]*PackageCompilation
	SourceFileSet map[string]*FileCompilation

	helper         *analyzer.AstHelper
	dependImporter func(importPath string, pkgBaseDir string) (*PackageCompilation, error)
}

// NewPackageCompilation creates a PackageCompilation using
// - pkg -  type:build.Package, result of go/build.Import or go list cmd,
// - depOnly - whether pkg is a depend-only package
// - importDependFn - help to import depend-only package
func NewPackageCompilation(pkg *build.Package, depOnly bool, importDependFn func(importPath string, pkgBaseDir string) (*PackageCompilation, error)) *PackageCompilation {
	pc := &PackageCompilation{
		ImportPath:     pkg.ImportPath,
		PackagePath:    util.RepoForPackage(pkg),
		DepOnly:        depOnly,
		BuiledPackage:  pkg,
		dependImporter: importDependFn,
		Deps:           make(map[string]*PackageCompilation),
		SourceFileSet:  make(map[string]*FileCompilation),
	}

	//log.Printf("new package %s %+v", pc.ImportPath, pc.PackagePath)

	return pc
}

// clear source files and dependency packages data
// which are compile runtime data
func (pcu *PackageCompilation) initial() {
	pcu.Deps = make(map[string]*PackageCompilation)
	pcu.SourceFileSet = make(map[string]*FileCompilation)
}

func (pcu *PackageCompilation) Compile() error {
	// initial compilation
	pcu.initial()

	fetcher, err := pcu.load()
	if err != nil {
		return err
	}

	return pcu.compile(fetcher)
}

// RunAnalyze helps analyzer to traverse and analyze source file
func (pcu *PackageCompilation) ForEach(fn func(*FileCompilation, *analyzer.AstHelper)) {
	for _, file := range pcu.SourceFileSet {
		fn(file, pcu.helper)
	}
}

// load imports all source files and dependency package objects
func (pcu *PackageCompilation) load() (Fetcher, error) {
	// load source files
	pcu.loadSourceFiles()

	// load source deps
	return pcu.loadDepPkgs()
}

// loadSourceFiles loads all source files
func (pcu *PackageCompilation) loadSourceFiles() {
	baseDir := pcu.BuiledPackage.Dir
	rootDir := pcu.BuiledPackage.Root
	for _, fileName := range pcu.BuiledPackage.GoFiles {
		filePath := ComputeFilePath(rootDir, baseDir, fileName)
		fc := NewFileCompilation(pcu.PackagePath, filePath)
		pcu.SourceFileSet[fileName] = fc
	}
}

// loadDepPkgs loads all dependency package objects
func (pcu *PackageCompilation) loadDepPkgs() (Fetcher, error) {
	baseDir := pcu.BuiledPackage.Dir
	deps := pcu.BuiledPackage.Imports
	var missing []string
	fetcher := make(mapFetcher)

	for _, depName := range deps {
		if depName == "unsafe" {
			// package unsafe is intrinsic; nothing to do
		} else if dep, err := pcu.dependImporter(depName, baseDir); err != nil || dep.BuiledPackage.PkgObj == "" {
			missing = append(missing, depName)
			log.Printf("miss dep for base dir %+v depend import path %+v", baseDir, depName)
		} else if _, ok := pcu.Deps[dep.ImportPath]; !ok {
			bp := dep.BuiledPackage
			path := bp.PkgObj // package object absolute path

			pcu.Deps[dep.ImportPath] = dep
			fd, err := FetchFileData(path)
			if err != nil {
				return nil, fmt.Errorf("fetch dependency package object %s: %v", path, err)
			}
			fetcher[path] = fd.Content
			//log.Printf("load dep %s, repo %s ,path %s", depName, dep.ImportPath, path)
		}
	}

	if len(missing) != 0 {
		return nil, &CompileMissingError{pcu.ImportPath, missing}
	}

	return fetcher, nil
}

// compile helps to parse source file to file ast and use a type check to resolve all type reference
func (pcu *PackageCompilation) compile(fetcher Fetcher) error {
	fset := token.NewFileSet()              // location info for the parser
	floc := make(map[*token.File]*ast.File) // file → ast
	depPathMap := make(map[string]string)   // import path → package object absolute path
	deps := make(map[string]*types.Package) // import path → package
	astFiles := make([]*ast.File, 0, 1)     // parsed sources

	for _, fi := range pcu.SourceFileSet {
		parsed, err := fi.Compile(fset)
		if err != nil {
			return err
		}
		astFiles = append(astFiles, parsed)
	}

	for _, dep := range pcu.Deps {
		depPathMap[dep.ImportPath] = dep.BuiledPackage.PkgObj
	}

	// Populate the location mapping. This relies on the fact that Iterate
	// reports its files in the order they were added to the set, which in turn
	// is their order in the files list.
	i := 0
	fset.Iterate(func(f *token.File) bool {
		floc[f] = astFiles[i]
		i++
		return true
	})

	var (
		err         error
		compileErrs []error
	)
	pi := &packageImporter{
		deps:       deps,
		fileSet:    fset,
		depPathMap: depPathMap,
		fetcher:    fetcher,
	}
	c := &types.Config{
		FakeImportC:              true, // so we can handle cgo
		DisableUnusedImportCheck: true, // this is not fatal to type-checking
		Importer:                 pi,
		Error:                    func(err error) { compileErrs = append(compileErrs, err) },
	}

	typeinfo := NewTypeInfo()
	pkg, err := c.Check(astFiles[0].Name.Name, fset, astFiles, typeinfo)
	for i, cerr := range compileErrs {
		log.Printf("compiling package error %d -  %s", i, cerr)
	}
	if err != nil {
		return err
	}
	pcu.helper = analyzer.NewAstHelper(pkg, fset, typeinfo)

	return nil
}

// Clone clones a PackageCompilation
// initial source files and dependency packages data which are compile runtime data
func (pcu *PackageCompilation) Clone() *PackageCompilation {
	return &PackageCompilation{
		ImportPath:     pcu.ImportPath,
		PackagePath:    pcu.PackagePath,
		DepOnly:        pcu.DepOnly,
		BuiledPackage:  pcu.BuiledPackage,
		dependImporter: pcu.dependImporter,
		Deps:           make(map[string]*PackageCompilation),
		SourceFileSet:  make(map[string]*FileCompilation),
	}
}

// GetPackagePath return the package path
func (pcu *PackageCompilation) GetPackagePath() *logpattern.PackagePath {
	return pcu.PackagePath
}

// packageImporter implements the types.Importer interface by fetching files
// from required inputs of a package compilation.
type packageImporter struct {
	deps       map[string]*types.Package // packages already loaded
	fileSet    *token.FileSet            // source location information
	depPathMap map[string]string         // import path → package object absolute path
	fetcher    Fetcher                   // access to required input contents
}

// Import satisfies the types.Importer interface using the captured data from
// the compilation unit.
func (pi *packageImporter) Import(importPath string) (*types.Package, error) {
	if pkg := pi.deps[importPath]; pkg != nil && pkg.Complete() {
		return pkg, nil
	} else if importPath == "unsafe" {
		// The "unsafe" package is special, and isn't usually added by the
		// resolver into the dependency map.
		pi.deps[importPath] = types.Unsafe
		return types.Unsafe, nil
	}

	// Fetch the required input holding the package for this import path, and
	// load its export data for use by the type resolver.
	path, ok := pi.depPathMap[importPath]
	if !ok {
		return nil, fmt.Errorf("package %s not found", importPath)
	}

	data, err := pi.fetcher.Fetch(path, importPath)
	if err != nil {
		return nil, fmt.Errorf("fetching %s(%s): %v", importPath, path, err)
	}
	r, err := gcexportdata.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("reading export data in %s(%s): %v", importPath, path, err)
	}
	return gcexportdata.Read(r, pi.fileSet, pi.deps, importPath)
}

// CompileMissingError is the concrete type of errors about missing dependencies.
type CompileMissingError struct {
	Path    string   // The import path of the incomplete package
	Missing []string // The import paths of the missing dependencies
}

func (m *CompileMissingError) Error() string {
	return fmt.Sprintf("package %q is missing %d imports (%s)",
		m.Path, len(m.Missing), strings.Join(m.Missing, ", "))
}

// NewTypeInfo creates a new types.Info value with empty maps for each of the
// fields needed for cross-reference indexing.
func NewTypeInfo() *types.Info {
	return &types.Info{
		Types:     make(map[ast.Expr]types.TypeAndValue),
		Defs:      make(map[*ast.Ident]types.Object),
		Uses:      make(map[*ast.Ident]types.Object),
		Implicits: make(map[ast.Node]types.Object),
	}
}

// A Fetcher retrieves the contents of a file given its path and/or hex-encoded
// SHA256 digest, at least one of which must be set.
// TODO: put it into package compiler
type Fetcher interface {
	Fetch(path, digest string) ([]byte, error)
}

type mapFetcher map[string][]byte

// Fetch implements the analysis.Fetcher interface.
// The digest argument is ignored.
func (m mapFetcher) Fetch(path, _ string) ([]byte, error) {
	if data, ok := m[path]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
}
