package util

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	. "github.com/pingcap/check"
)

func TestClient(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&testLogExtractorSuite{})

type testLogExtractorSuite struct {
}

func (t *testLogExtractorSuite) TestLogPatternRule(c *C) {
	rule := &logpattern_go_proto.LogPatternRule{}
	err := StrictDecodeFile("../../test/extractor/rule.example.cfg", rule)
	c.Assert(err, IsNil)
	c.Assert(rule.LogLevel, DeepEquals, []string{"fatal", "error"})

	// empty rule the return true
	res := MatchLogPatternRule(nil, "error", "")
	c.Assert(res, IsTrue)
	// empty rule the return true
	res = MatchLogPatternRule(nil, "", "")
	c.Assert(res, IsTrue)
	// misatch log level
	res = MatchLogPatternRule(rule, "warn", "")
	c.Assert(res, IsFalse)
	// misatch log level
	res = MatchLogPatternRule(rule, "", "")
	c.Assert(res, IsFalse)
	// matched log level
	res = MatchLogPatternRule(rule, "Error", "")
	c.Assert(res, IsTrue)
	// matched log level
	res = MatchLogPatternRule(rule, "Fatal", "")
	c.Assert(res, IsTrue)
}

func (t *testLogExtractorSuite) TestSaveAndLoadPatternRule(c *C) {

	tmpdir, err := ioutil.TempDir("./", "logpattern_test")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpdir)

	db, err := leveldb.Open(tmpdir, nil)
	c.Assert(err, IsNil)
	store := keyvalue.NewLogPatternStore(db)

	key, _ := keyvalue.EncodeLogPatternRuleKey()
	wr, err := db.Writer(context.Background())
	c.Assert(err, IsNil)

	// write error log pattern rule
	err = wr.Write(key, []byte("xxx"))
	c.Assert(err, IsNil)
	c.Assert(wr.Close(), IsNil)

	rule, err := GetLogPatternRule(store)
	c.Assert(err, NotNil)
	c.Assert(rule, IsNil)

	store.WriteLogPatternRule(context.Background(), &logpattern_go_proto.LogPatternRule{
		LogLevel: []string{"error"},
	})
	rule, err = GetLogPatternRule(store)
	c.Assert(err, IsNil)
	c.Assert(rule, DeepEquals, &logpattern_go_proto.LogPatternRule{
		LogLevel: []string{"error"},
	})

}
