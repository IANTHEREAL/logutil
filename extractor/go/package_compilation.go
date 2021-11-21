package extractor_go

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"log"
	"os"
	"strings"

	proto_common "github.com/IANTHEREAL/logutil/pkg/proto/common_go"
	"github.com/IANTHEREAL/logutil/pkg/util"
	"golang.org/x/tools/go/gcexportdata"
)

type PackageCompilation struct {
	ctx           build.Context
	Name          proto_common.XVName
	Repo          *util.RepoPath
	ImportPath    string
	DepOnly       bool
	BuiledPackage *build.Package
	SourceFiles   map[string]*FileInfo // file name => file info
	Deps          map[string]*PackageCompilation
	FileSet       map[string]*FileCompilation

	dependFinder func(importPath string, pkgBaseDir string) (*PackageCompilation, error)
}

func NewPackageCompilation(pkg *build.Package, depOnly bool, fn func(string, pkgBaseDir string) (*PackageCompilation, error)) *PackageCompilation {
	unit := &PackageCompilation{
		ImportPath:    pkg.ImportPath,
		Repo:          util.RepoForPackage(pkg),
		DepOnly:       depOnly,
		BuiledPackage: pkg,
		SourceFiles:   make(map[string]*FileInfo),
		Deps:          make(map[string]*PackageCompilation),
		dependFinder:  fn,
	}

	log.Printf("unit repo %s %+v", unit.ImportPath, unit.Repo)

	return unit
}

func (pcu *PackageCompilation) Clone() *PackageCompilation {
	return &PackageCompilation{
		ImportPath: pcu.ImportPath,
		Repo: &util.RepoPath{
			Repo: pcu.Repo.Repo,
			Root: pcu.Repo.Root,
			Path: pcu.Repo.Path,
		},
		DepOnly:       pcu.DepOnly,
		BuiledPackage: pcu.BuiledPackage,
		SourceFiles:   make(map[string]*FileInfo),
		Deps:          make(map[string]*PackageCompilation),
		dependFinder:  pcu.dependFinder,
	}
}

func (pcu *PackageCompilation) Resolve() error {
	// add source files
	pcu.addSourceFiles()
	// add source deps
	missing := pcu.addDeps()
	if len(missing) != 0 {
		return &CompileMissingError{pcu.ImportPath, missing}
	}

	fetcher, err := pcu.load()
	if err != nil {
		return err
	}

	err = pcu.compile(fetcher)
	if err != nil {
		return err
	}

	pcu.analyze()
	return nil
}

func (pcu *PackageCompilation) addSourceFiles() {
	baseDir := pcu.BuiledPackage.Dir
	rootDir := pcu.BuiledPackage.Root
	//log.Printf("add source files root %+v  base dir%+v files %+v", rootDir, baseDir, pcu.BuiledPackage.GoFiles)
	for _, fileName := range pcu.BuiledPackage.GoFiles {
		fi := NewFileInfo(rootDir, baseDir, fileName, pcu.Repo)
		pcu.SourceFiles[fileName] = fi
		//log.Printf("add source files root %+v  base dir%+v files %+v", rootDir, baseDir, fi)
	}
}

func (pcu *PackageCompilation) addDeps() []string {
	baseDir := pcu.BuiledPackage.Dir
	deps := pcu.BuiledPackage.Imports
	var missing []string

	for _, depName := range deps {
		if depName == "unsafe" {
			// package unsafe is intrinsic; nothing to do
		} else if unit, err := pcu.dependFinder(depName, baseDir); err != nil || unit.BuiledPackage.PkgObj == "" {
			missing = append(missing, depName)
			log.Printf("miss deps base dir %+v depend import path %+v", baseDir, depName)
		} else if _, ok := pcu.Deps[unit.ImportPath]; !ok {
			unit.SourceFiles[unit.BuiledPackage.PkgObj] = NewFileInfo(unit.BuiledPackage.Root, "", unit.BuiledPackage.PkgObj, pcu.Repo)
			pcu.Deps[unit.ImportPath] = unit
			log.Printf("add dep root %s files %+v", depName, unit.SourceFiles[unit.BuiledPackage.PkgObj])
		}
	}

	return missing
}

func (pcu *PackageCompilation) load() (mapFetcher, error) {
	fetcher := make(mapFetcher)
	//log.Printf("\n\n\n************** pkg %+v %+v", pcu.ImportPath, pcu.Repo)
	// import path: github.com/IANTHEREAL/logutil/pkg/util repo: &{Repo:https://github.com/IANTHEREAL/logutil Root:github.com/IANTHEREAL/logutil Path:pkg/util}

	filesInfo := make([]*FileInfo, 0, len(pcu.SourceFiles)+len(pcu.Deps))
	for _, fi := range pcu.SourceFiles {
		filesInfo = append(filesInfo, fi)
	}
	for _, dep := range pcu.Deps {
		for _, fi := range dep.SourceFiles {
			filesInfo = append(filesInfo, fi)
		}
	}

	// Ensure all the file contents are loaded, and update the digests.
	for _, fi := range filesInfo {
		log.Printf("start fetch digest %s , path %s", fi.Digest, fi.Path)
		/*
			    sources file
			    start deigest /Users/ianz/Work/go/src/github.com/pingcap/dm/dumpling/dumpling.go , path dumpling/dumpling.go
			    end deigest af0483b2da8d7e47784bcb90604b113c3a8422818d7f73038f9a0a60b57c2b1b , path dumpling/dumpling.go

				builtin pkg
				start deigest /usr/local/go/pkg/darwin_amd64/context.a , path pkg/darwin_amd64/context.a
				end deigest 8cd5bdbbd9eca1f15c8d408c1af7e790c62407a75057cdd4d77b34ee9489a44c , path pkg/darwin_amd64/context.a

				other pkg
				start deigest /Users/ianz/Library/Caches/go-build/43/43478aceaefb6d29989f4de0f5017fae40398a9afb1e5233af73e7259f45f12b-d ,
				path /Users/ianz/Library/Caches/go-build/43/43478aceaefb6d29989f4de0f5017fae40398a9afb1e5233af73e7259f45f12b-d

				end deigest 43478aceaefb6d29989f4de0f5017fae40398a9afb1e5233af73e7259f45f12b ,
				path /Users/ianz/Library/Caches/go-build/43/43478aceaefb6d29989f4de0f5017fae40398a9afb1e5233af73e7259f45f12b-d
		*/
		if !strings.Contains(fi.Digest, "/") {
			continue // skip those that are already complete
		}
		rc, err := os.Open(fi.Digest)
		if err != nil {
			return nil, fmt.Errorf("opening input %s: %v", fi.Digest, err)
		}
		fd, err := FetchFileData(fi.Path, rc)
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("fetch file %s: %v", fi.Digest, err)
		}
		fi.Digest = fd.Info.Digest
		fetcher[fi.Digest] = fd.Content
	}

	return fetcher, nil
}

func (pcu *PackageCompilation) compile(fetcher mapFetcher) error {
	fset := token.NewFileSet()              // location info for the parser
	astMap := make(map[*ast.File]*FileInfo) // ast file → file info
	floc := make(map[*token.File]*ast.File) // file → ast
	fmap := make(map[string]*FileInfo)      // import path → file info
	deps := make(map[string]*types.Package) // :: import path → package
	astFiles := make([]*ast.File, 0, 1)     // parsed sources

	for _, fi := range pcu.SourceFiles {
		path := fi.Path
		data, err := fetcher.Fetch(path, fi.Digest)
		if err != nil {
			return fmt.Errorf("fetching %q (%s): %v", path, fi.Digest, err)
		}

		parsed, err := parser.ParseFile(fset, path, data, parser.AllErrors|parser.ParseComments)
		if err != nil {
			return fmt.Errorf("parsing %q: %v", path, err)
		}
		astFiles = append(astFiles, parsed)
		astMap[parsed] = fi
	}

	for _, dep := range pcu.Deps {
		for _, fi := range dep.SourceFiles {
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

	analyzer := &AstAnalyzer{
		TypesInfo:    NewTypeInfo(),
		Functions:    make(map[ast.Node]string),
		PackageInits: make(map[*ast.File]string),
	}
	analyzer.Package, err = c.Check(astFiles[0].Name.Name, fset, astFiles, analyzer.TypesInfo)
	for i, cerr := range compileErrs {
		log.Printf("compiling package error %d -  %s", i, cerr)
	}
	if err != nil {
		return err
	}

	pcu.FileSet = make(map[string]*FileCompilation)
	for _, astFile := range astFiles {
		fi := astMap[astFile]
		if fi == nil {
			return fmt.Errorf("not found file info for ast file %s", astFile.Name)
		}
		pcu.FileSet[fi.Digest] = &FileCompilation{
			file:   fi,
			fAst:   astFile,
			fToken: fset,
			ai:     analyzer,
		}
	}

	return err
}

func (pcu *PackageCompilation) analyze() {
	for name, fc := range pcu.FileSet {
		log.Printf("ast name %s", name)
		fc.Analyze()
	}
}

// packageImporter implements the types.Importer interface by fetching files
// from the required inputs of a compilation unit.
type packageImporter struct {
	deps    map[string]*types.Package // packages already loaded
	fileSet *token.FileSet            // source location information
	fileMap map[string]*FileInfo      // :: import path → required input location
	fetcher Fetcher                   // access to required input contents
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
	fi := pi.fileMap[importPath]
	if fi == nil {
		return nil, fmt.Errorf("package %q not found", importPath)
	}

	data, err := pi.fetcher.Fetch(fi.Path, fi.Digest)
	if err != nil {
		return nil, fmt.Errorf("fetching %q (%s): %v", fi.Path, fi.Digest, err)
	}
	r, err := gcexportdata.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("reading export data in %q (%s): %v", fi.Path, fi.Digest, err)
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
