package util

import (
	"errors"
	"go/build"
	"regexp"
	"strings"
)

type vcsPath struct {
	prefix string // prefix this description applies to
	re     string // pattern for import path
	repo   string // repository to use (expand with match of re)
	vcs    string // version control system to use (expand with match of re)

	regexp *regexp.Regexp // cached compiled form of re
}

var srv = &vcsPath{
	prefix: "github.com/",
	re:     `^(?P<root>github\.com/[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+)(/[\p{L}0-9_.\-]+)*$`,
	repo:   "https://{root}",
}

func init() {
	srv.regexp = regexp.MustCompile(srv.re)
}

var (
	errUnknownSite = errors.New("dynamic lookup required to find mapping")
	golangCorpus   = "golang.org"
)

type RepoPath struct {
	// Repo is the repository URL, including scheme.
	Repo string

	// Root is the import path corresponding to the root of the
	// repository.
	Root string

	// import path relative to repo root
	Path string
}

func VCSRepoPath(importPath string) (*RepoPath, error) {
	m := srv.regexp.FindStringSubmatch(importPath)
	if m == nil {
		return nil, errUnknownSite
	}

	// Build map of named subexpression matches for expand.
	match := map[string]string{
		"prefix": srv.prefix,
	}
	for i, name := range srv.regexp.SubexpNames() {
		if name != "" && match[name] == "" {
			match[name] = m[i]
		}
	}

	if srv.repo != "" {
		match["repo"] = expand(match, srv.repo)
	}

	return &RepoPath{
		Repo: match["repo"],
		Root: match["root"],
		Path: strings.TrimPrefix(strings.TrimPrefix(importPath, match["root"]), "/"),
	}, nil
}

func RepoForPackage(bp *build.Package) *RepoPath {
	importPath := bp.ImportPath
	if r, err := VCSRepoPath(importPath); err == nil {
		return r
	}

	r := &RepoPath{}
	r.Path = bp.ImportPath
	if bp.Goroot {
		// This is a Go standard library package. By default the corpus is
		// implied to be "golang.org", but can be configured to use the default
		// corpus instead.
		r.Repo = golangCorpus
		r.Root = golangCorpus
	} else if strings.HasPrefix(importPath, ".") {
		// Local import; no corpus
	} else if i := strings.Index(importPath, "/"); i > 0 {
		// Take the first slash-delimited component to be the corpus.
		// e.g., import "foo/bar/baz" ⇒ corpus "foo", signature "bar/baz".
		r.Repo = importPath[:i]
		r.Root = importPath[:i]
		r.Path = importPath[i+1:]
	}

	return r
}

// ForBuiltin returns a VName for a Go built-in with the given signature.
func RepoForBuiltin(signature string) *RepoPath {
	return &RepoPath{
		Repo: golangCorpus,
		Root: "ref/spec",
		Path: signature,
	}
}

// expand rewrites s to replace {k} with match[k] for each key k in match.
func expand(match map[string]string, s string) string {
	for k, v := range match {
		s = strings.Replace(s, "{"+k+"}", v, -1)
	}
	return s
}
