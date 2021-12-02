package util

import (
	"go/build"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	. "github.com/pingcap/check"
)

var _ = Suite(&testRepoPathSuite{})

type testRepoPathSuite struct {
}

func (t *testRepoPathSuite) TestRepoForPackage(c *C) {
	// resolve package
	bp, err := build.Default.Import("github.com/pingcap/ticdc/dm/dm/common", "", build.AllowBinary)
	c.Assert(err, IsNil)
	c.Assert(bp.ImportPath, Equals, "github.com/pingcap/ticdc/dm/dm/common")

	packagePath := RepoForPackage(bp)
	c.Assert(packagePath.Repo, Equals, "github.com/pingcap/ticdc")
	c.Assert(packagePath.Path, Equals, "dm/dm/common")

	// resolve built-in package
	bp, err = build.Default.Import("fmt", "github.com/pingcap/ticdc/dm/dm/common", build.AllowBinary)
	c.Assert(err, IsNil)
	c.Assert(bp.ImportPath, Equals, "fmt")

	packagePath = RepoForPackage(bp)
	c.Assert(packagePath.Repo, Equals, "golang.org")
	c.Assert(packagePath.Path, Equals, "fmt")
}

func (t *testRepoPathSuite) TestVCSPath(c *C) {
	// resolve github path
	path, err := VCSPath("github.com/pingcap/ticdc/dm/dm/common")
	c.Assert(err, IsNil)
	c.Assert(path, DeepEquals, &logpattern_go_proto.PackagePath{
		Repo: "github.com/pingcap/ticdc",
		Path: "dm/dm/common",
	})

	// resolve gitlab repo
	path, err = VCSPath("gitlab.com/pingcap/ticdc/dm/dm/common")
	c.Assert(err, NotNil)
	c.Assert(path, IsNil)

	// resolve local package
	path, err = VCSPath("./")
	c.Assert(err, NotNil)
	c.Assert(path, IsNil)

	// resolve built-in package
	path, err = VCSPath("fmt")
	c.Assert(err, NotNil)
	c.Assert(path, IsNil)
}
