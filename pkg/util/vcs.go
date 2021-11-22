package util

import (
	"errors"
	"go/build"
	"regexp"
	"strings"
)

type vcsPath struct {
	re string // pattern for import path

	regexp *regexp.Regexp // cached compiled form of re
}

var githubRegex = &vcsPath{
	re: `^(?P<root>github\.com/[A-Za-z0-9_.\-]+/[A-Za-z0-9_.\-]+)(/[\p{L}0-9_.\-]+)*$`,
}

func init() {
	githubRegex.regexp = regexp.MustCompile(githubRegex.re)
}

var (
	errUnknownVCS = errors.New("not valid VCS path")
	golangCorpus  = "golang.org"
)

type RepoPath struct {
	// Repo is the root path of the repository.
	Repo string

	// import path relative to repo root
	Path string
}

// VCSPath resolves import path into {repo address(with schema), repo root path, import path relative to repo root}
func VCSPath(importPath string) (*RepoPath, error) {
	m := githubRegex.regexp.FindStringSubmatch(importPath)
	if m == nil {
		return nil, errUnknownVCS
	}

	// Build map of named subexpression matches for expand.
	match := make(map[string]string)
	for i, name := range githubRegex.regexp.SubexpNames() {
		if name != "" && match[name] == "" {
			match[name] = m[i]
		}
	}

	return &RepoPath{
		Repo: match["root"],
		Path: strings.TrimPrefix(strings.TrimPrefix(importPath, match["root"]), "/"),
	}, nil
}

// RepoForPackage resolves package path contains {repo address(with schema), repo root path, import path relative to repo root}
func RepoForPackage(bp *build.Package) *RepoPath {
	importPath := bp.ImportPath
	if r, err := VCSPath(importPath); err == nil {
		return r
	}

	r := &RepoPath{}
	r.Path = bp.ImportPath
	if bp.Goroot {
		// This is a Go standard library package. By default the corpus is
		// implied to be "golang.org", but can be configured to use the default
		// corpus instead.
		r.Repo = golangCorpus
	} else if strings.HasPrefix(importPath, ".") {
		// Local import; no corpus
	} else if i := strings.Index(importPath, "/"); i > 0 {
		// Take the first slash-delimited component to be the corpus.
		// e.g., import "foo/bar/baz" ⇒ repo "foo", path "bar/baz".
		r.Repo = importPath[:i]
		r.Path = importPath[i+1:]
	}

	return r
}

// expand rewrites s to replace {k} with match[k] for each key k in match.
func expand(match map[string]string, s string) string {
	for k, v := range match {
		s = strings.Replace(s, "{"+k+"}", v, -1)
	}
	return s
}
