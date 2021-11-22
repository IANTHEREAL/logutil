package extractor_go

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/IANTHEREAL/logutil/pkg/util"
)

/*
	    sources file
	    abs path /Users/ianz/Work/go/src/github.com/pingcap/dm/dumpling/dumpling.go
		relative ptah dumpling/dumpling.go
	    digest af0483b2da8d7e47784bcb90604b113c3a8422818d7f73038f9a0a60b57c2b1b

		builtin pkg
		abs path /usr/local/go/pkg/darwin_amd64/context.a
		relative path pkg/darwin_amd64/context.a
	    digest 8cd5bdbbd9eca1f15c8d408c1af7e790c62407a75057cdd4d77b34ee9489a44c

		other pkg
		abs path /Users/ianz/Library/Caches/go-build/43/43478aceaefb6d29989f4de0f5017fae40398a9afb1e5233af73e7259f45f12b-d
		relative path /Users/ianz/Library/Caches/go-build/43/43478aceaefb6d29989f4de0f5017fae40398a9afb1e5233af73e7259f45f12b-d
		digest 43478aceaefb6d29989f4de0f5017fae40398a9afb1e5233af73e7259f45f12b ,
*/
type FilePath struct {
	RelPath string
	AbsPath string
	Digest  string
}

func (f *FilePath) String() string {
	return fmt.Sprintf("file %s(%s - %s)", f.Digest, f.RelPath, f.AbsPath)
}

// FileCompilation
type FileCompilation struct {
	repo     *util.RepoPath
	filePath *FilePath
	fAst     *ast.File

	data []byte
}

// NewFileCompilation create a file Compilation
// use  rootDir, baseDir, fileName (from build.package)  to compute relative and absolute path
func NewFileCompilation(rootDir, baseDir, fileName string, repo *util.RepoPath) *FileCompilation {
	absPath := fileName
	if baseDir != "" {
		absPath = filepath.Join(baseDir, fileName)
	}
	relativePath := strings.TrimPrefix(strings.TrimPrefix(absPath, rootDir), "/")

	return &FileCompilation{
		repo: repo,
		filePath: &FilePath{
			RelPath: relativePath,
			AbsPath: absPath,
			Digest:  absPath,
		},
	}
}

func (fc *FileCompilation) Compile(fset *token.FileSet) (*ast.File, error) {
	fd, err := fc.FetchFileData()
	if err != nil {
		return nil, err
	}

	filePath := fc.filePath.RelPath
	parsed, err := parser.ParseFile(fset, fc.filePath.RelPath, fd.Content, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %v", filePath, err)
	}

	fc.fAst = parsed
	return parsed, nil
}

func (fc *FileCompilation) GetAst() *ast.File {
	return fc.fAst
}

func (fc *FileCompilation) GetPath() *FilePath {
	return fc.filePath
}

func (fc *FileCompilation) FetchFileData() (*FileData, error) {
	rc, err := os.Open(fc.filePath.AbsPath)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %v", fc.filePath, err)
	}
	defer rc.Close()

	var buf bytes.Buffer
	hash := sha256.New()

	w := io.MultiWriter(&buf, hash)
	if _, err := io.Copy(w, rc); err != nil {
		return nil, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	fc.filePath.Digest = digest
	return &FileData{
		Content: buf.Bytes(),
		Digest:  digest,
	}, nil
}

func (fc *FileCompilation) RunAnalyze(ai Aanalyzer, helper *AstHelper) {
	//log.Printf("analyze %s", fc.filePath)
	ai.Run(fc.fAst, helper)
}

type FileData struct {
	Content []byte
	Digest  string
}
