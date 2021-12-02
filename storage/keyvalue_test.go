package keyvalue

import (
	"context"
	"io/ioutil"
	"os"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	. "github.com/pingcap/check"
)

var _ = Suite(&testKeyValueSuite{})

type testKeyValueSuite struct {
}

func (t *testKeyValueSuite) TestRepoForPackage(c *C) {
	tmpdir, err := ioutil.TempDir("./", "logpattern_test")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpdir)

	db, err := leveldb.Open(tmpdir, nil)
	c.Assert(err, IsNil)
	store := keyvalue.NewLogPatternStore(db)

	// test log pattern
	pattern := &logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master3/config.go",
			LineNumber:   154,
			ColumnOffset: 7274,
		},
		Func: &logpattern_go_proto.FuncInfo{
			Name: "Toml",
			Pos: &logpattern_go_proto.Position{
				FilePath:     "dm/dm/master/config.go",
				LineNumber:   138,
				ColumnOffset: 7140,
			},
		},
		Level:     "error",
		Signature: []string{"\"fail to marshal config to % wwwww"},
	}

	err = store.WriteLogPattern(context.Background(), pattern)
	c.Assert(err, NotNil)

	retPattern := &logpattern_go_proto.LogPattern{}
	err = store.ScanLogPattern(context.Background(), func(_, value []byte) error {
		return retPattern.Unmarshal(value)
	})
	c.Assert(err, IsNil)
	c.Assert(retPattern, DeepEquals, pattern)

	// write error log pattern
	patternKey, err := keyvalue.EncodeLogKey(pattern.Pos)
	c.Assert(err, IsNil)
	wr, err := db.Writer(context.Background())
	c.Assert(err, IsNil)
	err = wr.Write(patternKey, []byte("xxx"))
	c.Assert(err, IsNil)
	c.Assert(wr.Close(), IsNil)

	err = store.ScanLogPattern(context.Background(), func(_, value []byte) error {
		return retPattern.Unmarshal(value)
	})
	c.Assert(err, NotNil)

	// test log coverage
	coverage := &logpattern_go_proto.Coverage{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master3/config.go",
			LineNumber:   154,
			ColumnOffset: 7274,
		},
		CovCount: 32,
		CovCountByLog: map[string]int32{
			"xxx": 32,
		},
	}

	err = store.WriteLogCoverage(context.Background(), coverage)
	c.Assert(err, NotNil)

	retCoverage := &logpattern_go_proto.Coverage{}
	err = store.ScanLogCoverage(context.Background(), func(_, value []byte) error {
		return retCoverage.Unmarshal(value)
	})
	c.Assert(err, IsNil)
	c.Assert(retCoverage, DeepEquals, coverage)

	// write error log coverage
	coverageKey, err := keyvalue.EncodeCoverageKey(coverage.Pos)
	c.Assert(err, IsNil)
	wr, err = db.Writer(context.Background())
	c.Assert(err, IsNil)
	err = wr.Write(coverageKey, []byte("xxx"))
	c.Assert(err, IsNil)
	c.Assert(wr.Close(), IsNil)

	err = store.ScanLogCoverage(context.Background(), func(_, value []byte) error {
		return retCoverage.Unmarshal(value)
	})
	c.Assert(err, NotNil)
}
