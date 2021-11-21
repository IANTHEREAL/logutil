package extractor_go

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"go/ast"
	"go/token"
	"io"
	"path/filepath"
	"strings"

	"github.com/IANTHEREAL/logutil/pkg/util"
)

type FileInfo struct {
	Repo   *util.RepoPath
	Path   string
	Digest string
}

func NewFileInfo(rootDir, baseDir, file string, repo *util.RepoPath) *FileInfo {
	if !strings.HasSuffix(rootDir, "/") {
		rootDir += "/"
	}
	path := file
	if baseDir != "" {
		path = filepath.Join(baseDir, file)
	}
	relativePath := strings.TrimPrefix(path, rootDir)
	return &FileInfo{
		Path:   relativePath,
		Digest: path,
		Repo:   repo,
	}
}

type FileCompilation struct {
	file *FileInfo

	fAst   *ast.File
	fToken *token.FileSet

	ai *AstAnalyzer
}

func (fc *FileCompilation) Analyze() {
	ai := fc.ai
	ast.Walk(newASTVisitor(func(node ast.Node, stack stackFunc) bool {
		switch n := node.(type) {
		case *ast.Ident:
		//pcu.visitIdent(n, stack)
		case *ast.FuncDecl:
			ai.visitFuncDecl(n, stack)
		case *ast.FuncLit:
			ai.visitFuncLit(n, stack)
		case *ast.BasicLit:
			ai.isLog(n, stack)
		}
		return true
	}), fc.fAst)
}

type FileData struct {
	Content []byte    `protobuf:"bytes,1,opt,name=content,proto3" json:"content,omitempty"`
	Info    *FileInfo `protobuf:"bytes,2,opt,name=info,proto3" json:"info,omitempty"`
	Missing bool      `protobuf:"varint,3,opt,name=missing,proto3" json:"missing,omitempty"`
}

// FetchFileData creates a file data protobuf message by fully reading the contents
// of r, having the designated path.
func FetchFileData(path string, r io.Reader) (*FileData, error) {
	var buf bytes.Buffer
	hash := sha256.New()

	w := io.MultiWriter(&buf, hash)
	if _, err := io.Copy(w, r); err != nil {
		return nil, err
	}
	digest := hex.EncodeToString(hash.Sum(nil))
	return &FileData{
		Content: buf.Bytes(),
		Info: &FileInfo{
			Path:   path,
			Digest: digest,
		},
	}, nil
}
