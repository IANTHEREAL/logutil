package extractor_go

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

	"github.com/IANTHEREAL/logutil/pkg/util"
	"golang.org/x/tools/go/gcexportdata"
)

// PackageCompilation helps to compile package to get AST set and type use info
// usage:
//  compilation := NewPackageCompilation(build.Package, depOnly, importDependFn)
//  err := compilation.Compile()   // compile package to get file AST
//  ...
//  fileCompilation := compilation.GetFileCompilation() // get all file compilation with file AST
//  fileCompilation.Aanalyze(astAnalyzer)
//  or
//  compilation.EachFileCompilation(astAnalyzer) // traverse file ast and analyze at the same time

type PackageCompilation struct {
	// read only
	ImportPath    string
	Repo          *util.RepoPath
	BuiledPackage *build.Package
	DepOnly       bool

	// written at compiling
	Deps          map[string]*PackageCompilation
	SourceFileSet map[string]*FileCompilation

	helper         *AstHelper
	dependImporter func(importPath string, pkgBaseDir string) (*PackageCompilation, error)
}

func NewPackageCompilation(pkg *build.Package, depOnly bool, fn func(importPath string, pkgBaseDir string) (*PackageCompilation, error)) *PackageCompilation {
	pc := &PackageCompilation{
		ImportPath:     pkg.ImportPath,
		Repo:           util.RepoForPackage(pkg),
		DepOnly:        depOnly,
		BuiledPackage:  pkg,
		dependImporter: fn,
		Deps:           make(map[string]*PackageCompilation),
		SourceFileSet:  make(map[string]*FileCompilation),
	}

	log.Printf("new package %s %+v", pc.ImportPath, pc.Repo)

	return pc
}

func (pcu *PackageCompilation) Clone() *PackageCompilation {
	return &PackageCompilation{
		ImportPath:     pcu.ImportPath,
		Repo:           pcu.Repo,
		DepOnly:        pcu.DepOnly,
		BuiledPackage:  pcu.BuiledPackage,
		dependImporter: pcu.dependImporter,
		Deps:           make(map[string]*PackageCompilation),
		SourceFileSet:  make(map[string]*FileCompilation),
	}
}

// clear compilation information and initial related structure
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

func (pcu *PackageCompilation) RunAnalyze(ai Aanalyzer) {
	for _, file := range pcu.SourceFileSet {
		file.RunAnalyze(ai, pcu.helper)
	}
}

func (pcu *PackageCompilation) load() (Fetcher, error) {
	// add source files
	pcu.addSourceFiles()

	// add source deps
	if err := pcu.addDepPkgs(); err != nil {
		return nil, err
	}

	return pcu.loadDepsContent()
}

func (pcu *PackageCompilation) addSourceFiles() {
	baseDir := pcu.BuiledPackage.Dir
	rootDir := pcu.BuiledPackage.Root
	for _, fileName := range pcu.BuiledPackage.GoFiles {
		fi := NewFileCompilation(rootDir, baseDir, fileName, pcu.Repo)
		pcu.SourceFileSet[fileName] = fi
	}
}

func (pcu *PackageCompilation) addDepPkgs() error {
	baseDir := pcu.BuiledPackage.Dir
	deps := pcu.BuiledPackage.Imports
	var missing []string

	for _, depName := range deps {
		if depName == "unsafe" {
			// package unsafe is intrinsic; nothing to do
		} else if dep, err := pcu.dependImporter(depName, baseDir); err != nil || dep.BuiledPackage.PkgObj == "" {
			missing = append(missing, depName)
			log.Printf("miss dep for base dir %+v depend import path %+v", baseDir, depName)
		} else if _, ok := pcu.Deps[dep.ImportPath]; !ok {
			bp := dep.BuiledPackage
			rootDir := bp.Root
			pkgObj := bp.PkgObj
			fc := NewFileCompilation(rootDir, "", bp.PkgObj, dep.Repo)
			dep.SourceFileSet[pkgObj] = fc
			pcu.Deps[dep.ImportPath] = dep
			log.Printf("add dep %s -- repo %s -- path %s", depName, dep.Repo, fc.GetPath())
		}
	}

	if len(missing) != 0 {
		return &CompileMissingError{pcu.ImportPath, missing}
	}

	return nil
}

func (pcu *PackageCompilation) loadDepsContent() (Fetcher, error) {
	fetcher := make(mapFetcher)

	// Ensure all the file contents are loaded, and update the digests.
	for name, dep := range pcu.Deps {
		for _, fi := range dep.SourceFileSet {
			filePath := fi.GetPath()
			log.Printf("start load dep package %s - %s", name, filePath)
			//todo: improve it for multiple packages
			if !strings.Contains(filePath.Digest, "/") {
				continue // skip those that are already complete
			}
			fd, err := fi.FetchFileData()
			if err != nil {
				return nil, fmt.Errorf("fetch file %s: %v", fi.GetPath(), err)
			}
			fetcher[fd.Digest] = fd.Content
		}
	}

	return fetcher, nil
}

func (pcu *PackageCompilation) compile(fetcher Fetcher) error {
	fset := token.NewFileSet()                // location info for the parser
	floc := make(map[*token.File]*ast.File)   // file → ast
	fmap := make(map[string]*FileCompilation) // import path → file info
	deps := make(map[string]*types.Package)   // import path → package
	astFiles := make([]*ast.File, 0, 1)       // parsed sources

	for _, fi := range pcu.SourceFileSet {
		parsed, err := fi.Compile(fset)
		if err != nil {
			return err
		}
		astFiles = append(astFiles, parsed)
	}

	for _, dep := range pcu.Deps {
		for _, fi := range dep.SourceFileSet {
			fmap[dep.ImportPath] = fi
		}
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
		deps:    deps,
		fileSet: fset,
		fileMap: fmap,
		fetcher: fetcher,
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
	pcu.helper = NewAstHelper(pkg, fset, typeinfo)

	return nil
}

/*func (pcu *PackageCompilation) analyze() {
	for name := range pcu.SourceFileSet {
		log.Printf("ast name %s", name)
		//fc.Analyze()
	}
}*/

// packageImporter implements the types.Importer interface by fetching files
// from the required inputs of a compilation unit.
type packageImporter struct {
	deps    map[string]*types.Package   // packages already loaded
	fileSet *token.FileSet              // source location information
	fileMap map[string]*FileCompilation // import path → required input location
	fetcher Fetcher                     // access to required input contents
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
	fc := pi.fileMap[importPath]
	if fc == nil {
		return nil, fmt.Errorf("package %q not found", importPath)
	}
	fi := fc.GetPath()

	data, err := pi.fetcher.Fetch(fi.RelPath, fi.Digest)
	if err != nil {
		return nil, fmt.Errorf("fetching %q (%s): %v", fi.RelPath, fi.Digest, err)
	}
	r, err := gcexportdata.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("reading export data in %q (%s): %v", fi.RelPath, fi.Digest, err)
	}
	return gcexportdata.Read(r, pi.fileSet, pi.deps, importPath)
}

// A Fetcher retrieves the contents of a file given its path and/or hex-encoded
// SHA256 digest, at least one of which must be set.
type Fetcher interface {
	Fetch(path, digest string) ([]byte, error)
}

type mapFetcher map[string][]byte

// Fetch implements the analysis.Fetcher interface. The path argument is ignored.
func (m mapFetcher) Fetch(_, digest string) ([]byte, error) {
	if data, ok := m[digest]; ok {
		return data, nil
	}
	return nil, os.ErrNotExist
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
