package matcher

import (
	"testing"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	. "github.com/pingcap/check"
)

func TestClient(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&testPatternTrieSuite{})

type testPatternTrieSuite struct {
}

type trieCase struct {
	input []string
	res   *MatchedResult
}

func (t *testPatternTrieSuite) TestMatch(c *C) {
	trie := NewPatternTrie()

	patterns, bps := testGenerateLogPattern()
	for _, pattern := range patterns {
		trie.Insert(pattern.Signature[0], pattern)
	}

	cases := []*trieCase{
		{
			input: []string{"", "error", "config.go:159"},
			res: &MatchedResult{
				Patterns: map[string]*BriefPattern{},
			},
		},
		{
			input: []string{"\"fail to marshal config to toml\"", "error", "config.go:159"},
			res: &MatchedResult{
				Patterns: map[string]*BriefPattern{bps[0].ID(): bps[0], bps[4].ID(): bps[4]},
			},
		},
		{
			input: []string{"\"fail to marshal config to toml\"", "error", "config.go:154"},
			res: &MatchedResult{
				Patterns: map[string]*BriefPattern{bps[3].ID(): bps[3], bps[1].ID(): bps[1], bps[5].ID(): bps[5]},
			},
		},
		{
			input: []string{"\"fail to marshal config xxx to toml\"", "error", "config.go:154"},
			res: &MatchedResult{
				Patterns: map[string]*BriefPattern{bps[1].ID(): bps[1]},
			},
		},
		{
			input: []string{"\"fail to marshal config  to toml\"", "warn", "config.go:159"},
			res: &MatchedResult{
				Patterns: map[string]*BriefPattern{bps[2].ID(): bps[2]},
			},
		},
		{
			input: []string{"\"fail to marshal config to toml\"xxxxx", "error", "config.go:159"},
			res: &MatchedResult{
				Patterns: map[string]*BriefPattern{bps[4].ID(): bps[4]},
			},
		},
		{
			input: []string{"\"fail to marshal config to %toml\"", "error", "config.go:159"},
			res: &MatchedResult{
				Patterns: map[string]*BriefPattern{bps[6].ID(): bps[6]},
			},
		},
	}

	for _, testCase := range cases {
		ret := trie.Match(testCase.input[0], testCase.input[1], testCase.input[2])
		c.Assert(ret.Patterns, HasLen, len(testCase.res.Patterns))
		for id, pattern := range ret.Patterns {
			bp := testCase.res.Patterns[id]
			c.Assert(bp, NotNil)
			c.Assert(pattern.ID(), Equals, bp.ID())
			c.Assert(pattern.matchedPos, Equals, bp.matchedPos)
			c.Assert(pattern.matchedLevel, Equals, bp.matchedLevel)
		}
	}
}

func (t *testPatternTrieSuite) TestBriefPattern(c *C) {
	patterns, bps := testGenerateLogPattern()

	for index, pattern := range patterns {
		bp := NewBriefPattern(pattern)
		c.Assert(bp.ID(), Equals, bps[index].ID())
		c.Assert(bp.matchedPos, Equals, bps[index].matchedPos)
		c.Assert(bp.matchedLevel, Equals, bps[index].matchedLevel)
	}

	bp := NewBriefPattern(&logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "config.go",
			LineNumber:   159,
			ColumnOffset: 7274,
		},
		Level: "warn",
	})
	c.Assert(bp.ID(), Equals, "github.com/pingcap/ticdc/dm:config.go:159:7274")
	c.Assert(bp.matchedPos, Equals, "config.go:159")
	c.Assert(bp.matchedLevel, Equals, "warn")
}

func (t *testPatternTrieSuite) TestMatchedResult(c *C) {
	res1 := newMatchedResult(&MatchedOptions{
		LogLevel: "error",
		Position: "db.go:181",
	})

	c.Assert(res1.empty(), IsTrue)

	patterns, _ := testGenerateLogPattern()
	bi := &baseItem{pattern: make(map[string]*BriefPattern)}
	for _, pattern := range patterns {
		bi.setLogPattern(pattern)
	}
	bi.setLogPattern(&logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master3/config.go",
			LineNumber:   159,
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
		Signature: []string{"\"fail to marshal config to %%toml\""},
	})
	c.Assert(bi.pattern, HasLen, len(patterns))

	res1.append(bi)
	c.Assert(res1.empty(), IsTrue)

	res2 := newMatchedResult(&MatchedOptions{
		LogLevel: "error",
		Position: "config.go:159",
	})
	res2.append(bi)
	c.Assert(res2.empty(), IsFalse)
	c.Assert(res2.Patterns, HasLen, 3)
}

type repalceFormatPlaceholderCase struct {
	input   string
	retPos  int
	retByte byte
}

func (t *testPatternTrieSuite) TestRepalceFormatPlaceholder(c *C) {
	testCases := []*repalceFormatPlaceholderCase{
		{ // empty should return 0, ""
			input:   "",
			retPos:  0,
			retByte: '*',
		},
		{
			input:   "%test",
			retPos:  1,
			retByte: '%',
		},
		{
			input:   "xxx",
			retPos:  1,
			retByte: '*',
		},
		{
			input:   "#.3xxx",
			retPos:  4,
			retByte: '*',
		},
		{
			input:   "www",
			retPos:  3,
			retByte: '*',
		},
	}

	for _, testCase := range testCases {
		pos, b := repalceFormatPlaceholder(testCase.input)
		c.Assert(pos, Equals, testCase.retPos)
		c.Assert(b, Equals, testCase.retByte)
	}
}

func testGenerateLogPattern() ([]*logpattern_go_proto.LogPattern, []*BriefPattern) {
	var patterns []*logpattern_go_proto.LogPattern

	/*
		<package_path:<repo:"github.com/pingcap/ticdc/dm" > file_path:"dm/dm/master/config.go" line_number:159 column_offset:7274 >
		func:<name:"Toml" pos:<file_path:"dm/dm/master/config.go" line_number:154 column_offset:7140 > > level:"error" signature:"\"fail to marshal config to toml\""
	*/
	pattern := &logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master/config.go",
			LineNumber:   159,
			ColumnOffset: 7274,
		},
		Func: &logpattern_go_proto.FuncInfo{
			Name: "Toml",
			Pos: &logpattern_go_proto.Position{
				FilePath:     "dm/dm/master/config.go",
				LineNumber:   154,
				ColumnOffset: 7140,
			},
		},
		Level:     "error",
		Signature: []string{"\"fail to marshal config to toml\""},
	}
	patterns = append(patterns, pattern)

	pattern = &logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master/config.go",
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
		Signature: []string{"\"fail to marshal config %#vto toml\""},
	}
	patterns = append(patterns, pattern)

	pattern = &logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master2/config.go",
			LineNumber:   159,
			ColumnOffset: 7274,
		},
		Func: &logpattern_go_proto.FuncInfo{
			Name: "Toml",
			Pos: &logpattern_go_proto.Position{
				FilePath:     "dm/dm/master/config.go",
				LineNumber:   154,
				ColumnOffset: 7140,
			},
		},
		Level:     "warn",
		Signature: []string{"\"fail to marshal config %s%s to toml\""},
	}
	patterns = append(patterns, pattern)

	pattern = &logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master2/config.go",
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
		Signature: []string{"\"fail to marshal config to toml\""},
	}
	patterns = append(patterns, pattern)

	pattern = &logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master3/config.go",
			LineNumber:   159,
			ColumnOffset: 7274,
		},
		Func: &logpattern_go_proto.FuncInfo{
			Name: "Toml",
			Pos: &logpattern_go_proto.Position{
				FilePath:     "dm/dm/master/config.go",
				LineNumber:   154,
				ColumnOffset: 7140,
			},
		},
		Level:     "error",
		Signature: []string{"\"fail to marshal config to toml\"%"},
	}
	patterns = append(patterns, pattern)

	pattern = &logpattern_go_proto.LogPattern{
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
	patterns = append(patterns, pattern)

	pattern = &logpattern_go_proto.LogPattern{
		Pos: &logpattern_go_proto.Position{
			PackagePath: &logpattern_go_proto.PackagePath{
				Repo: "github.com/pingcap/ticdc/dm",
			},
			FilePath:     "dm/dm/master4/config.go",
			LineNumber:   159,
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
		Signature: []string{"\"fail to marshal config to %%toml\""},
	}
	patterns = append(patterns, pattern)

	bps := []*BriefPattern{
		{id: "github.com/pingcap/ticdc/dm:dm/dm/master/config.go:159:7274", matchedPos: "config.go:159", matchedLevel: "error"},
		{id: "github.com/pingcap/ticdc/dm:dm/dm/master/config.go:154:7274", matchedPos: "config.go:154", matchedLevel: "error"},
		{id: "github.com/pingcap/ticdc/dm:dm/dm/master2/config.go:159:7274", matchedPos: "config.go:159", matchedLevel: "warn"},
		{id: "github.com/pingcap/ticdc/dm:dm/dm/master2/config.go:154:7274", matchedPos: "config.go:154", matchedLevel: "error"},
		{id: "github.com/pingcap/ticdc/dm:dm/dm/master3/config.go:159:7274", matchedPos: "config.go:159", matchedLevel: "error"},
		{id: "github.com/pingcap/ticdc/dm:dm/dm/master3/config.go:154:7274", matchedPos: "config.go:154", matchedLevel: "error"},
		{id: "github.com/pingcap/ticdc/dm:dm/dm/master4/config.go:159:7274", matchedPos: "config.go:159", matchedLevel: "error"},
	}

	return patterns, bps
}
