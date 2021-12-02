package cmd

import (
	"context"
	"io/ioutil"
	"os"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	. "github.com/pingcap/check"
)

var _ = Suite(&testLogExtractorSuite{})

type testLogExtractorSuite struct {
}

func (t *testLogExtractorSuite) TestLogExtract(c *C) {
	tmpdir, err := ioutil.TempDir("./", "logpattern_test")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpdir)

	db, err := leveldb.Open(tmpdir, nil)
	c.Assert(err, IsNil)
	store := keyvalue.NewLogPatternStore(db)

	ExtractLogPattern(store, "./", &logpattern_go_proto.LogPatternRule{
		LogLevel: []string{"fatal"},
	})

	count := 0
	store.ScanLogPattern(context.Background(), func(_, value []byte) error {
		lp := &logpattern_go_proto.LogPattern{}
		err1 := lp.Unmarshal(value)
		c.Assert(err1, IsNil)
		count++
		return err
	})
	c.Assert(count, Equals, 13)
}
