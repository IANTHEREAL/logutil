package cmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/IANTHEREAL/logutil/pkg/util"
	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	log_reporter "github.com/IANTHEREAL/logutil/reporter"
	log_scanner "github.com/IANTHEREAL/logutil/scanner"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	. "github.com/pingcap/check"
)

func TestClient(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&testLogScannerSuite{})

type testLogScannerSuite struct {
}

func (t *testLogScannerSuite) TestLogScan(c *C) {
	storePath := "../test/log_pattern"
	var logs []string

	err := filepath.Walk("../test/dm_logs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			if filepath.Ext(path) == ".log" {
				absPath, err := filepath.Abs(path)
				if err != nil {
					return err
				}
				logs = append(logs, absPath)
			}
		}
		return nil
	})
	c.Assert(err, IsNil)

	db, err := leveldb.Open(storePath, nil)
	c.Assert(err, IsNil)
	store := keyvalue.NewLogPatternStore(db)

	testLogScan(c, store, logs)
	testCoverage(c, store)
}

func testCoverage(c *C, store *keyvalue.Store) {
	cov, err := log_reporter.NewCoverager(store)
	c.Assert(err, IsNil)
	total, covCount := cov.OverallCoverage()
	c.Assert(total, Equals, 247)
	c.Assert(covCount, Equals, 13)
}

func testLogScan(c *C, store *keyvalue.Store, logs []string) {
	rule, err := util.GetLogPatternRule(store)
	c.Assert(err, IsNil)
	c.Assert(rule, DeepEquals, &logpattern_go_proto.LogPatternRule{
		LogLevel: []string{"error"},
	})

	p, err := log_scanner.NewLogProcessor(store, logs)
	c.Assert(err, IsNil)

	c.Assert(p.Run(context.Background(), rule), IsNil)
}
