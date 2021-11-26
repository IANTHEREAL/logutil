package compiler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/IANTHEREAL/logutil/extractor/go/analyzer"
	logppatern "github.com/IANTHEREAL/logutil/proto"
)

/*
    FilePath contains the absolute path, relative path, and file hex-encoded SHA256 digest of go source file

	for example:
	sources file
	abs path /Users/ianz/Work/go/src/github.com/pingcap/dm/dumpling/dumpling.go
	relative ptah dumpling/dumpling.go
	digest af0483b2da8d7e47784bcb90604b113c3a8422818d7f73038f9a0a60b57c2b1b
*/
type FilePath struct {
	RelPath string
	AbsPath string
	Digest  string
}

// ComputeFilePath computes file path using rootDir(BuiledPackage.Root), baseDir(BuiledPackage.Dir), source file name
func ComputeFilePath(rootDir, baseDir, fileName string) *FilePath {
	absPath := fileName
	if baseDir != "" {
		absPath = filepath.Join(baseDir, fileName)
	}
	relativePath := strings.TrimPrefix(strings.TrimPrefix(absPath, rootDir), "/")
	return &FilePath{
		RelPath: relativePath,
		AbsPath: absPath,
		Digest:  absPath,
	}
}

func (f *FilePath) String() string {
	return fmt.Sprintf("file %s(%s)", f.AbsPath, f.Digest)
}

// FileCompilation
// it is not concurrency safe
type FileCompilation struct {
	PackagePath *logppatern.PackagePath
	filePath    *FilePath
	fAst        *ast.File
}

// NewFileCompilation creates a file compilation represents a go source file that will be compiled
func NewFileCompilation(pkg *logppatern.PackagePath, filePath *FilePath) *FileCompilation {
	return &FileCompilation{
		PackagePath: pkg,
		filePath:    filePath,
	}
}

// Compile use go/parser to parse a go source file, rerurn the file ast
func (fc *FileCompilation) Compile(fset *token.FileSet) (*ast.File, error) {
	fd, err := FetchFileData(fc.filePath.AbsPath)
	if err != nil {
		return nil, err
	}
	fc.filePath.Digest = fd.Digest

	filePath := fc.filePath.RelPath
	parsed, err := parser.ParseFile(fset, filePath, fd.Content, parser.AllErrors|parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %v", filePath, err)
	}

	fc.fAst = parsed
	return parsed, nil
}

// RunAnalysis helps analyzer to analyze the file ast
func (fc *FileCompilation) RunAnalysis(ai analyzer.Aanalyzer, helper *analyzer.AstHelper) error {
	if fc.fAst == nil {
		return fmt.Errorf("please compile file %s before analyzing", fc.filePath)
	}

	log.Printf("analyze %s", fc.filePath)
	ai.Run(fc.fAst, helper)
	//log.Printf("done  %s", fc.filePath)
	return nil
}

type FileData struct {
	Content []byte
	Digest  string
}

// FetchFileData reads file data from source file or package object file (like .a file)
// usage：
// * fetch source file data to parse
// * fetch package object for package type checker to resolve type reference
func FetchFileData(path string) (*FileData, error) {
	rc, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file %s: %v", path, err)
	}
	defer rc.Close()

	var buf bytes.Buffer
	hash := sha256.New()

	w := io.MultiWriter(&buf, hash)
	if _, err := io.Copy(w, rc); err != nil {
		return nil, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	return &FileData{
		Content: buf.Bytes(),
		Digest:  digest,
	}, nil
}
