package matcher

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/IANTHEREAL/logutil/pkg/util"
	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	scanner "github.com/IANTHEREAL/logutil/scanner/log_scan"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

// BriefPattern only contains brief data of log pattern
type BriefPattern struct {
	// unique id generate by *logpattern_go_proto Position
	id string

	// only contains file name and line number, e.g. db.go:181
	matchedPos   string
	matchedLevel string

	pattern *logpattern_go_proto.LogPattern
}

// NewBriefPattern generate a brief pattern using *logpattern_go_proto Position
func NewBriefPattern(pattern *logpattern_go_proto.LogPattern) *BriefPattern {
	patternPos := pattern.GetPos()

	return &BriefPattern{
		id:           util.PosToStr(patternPos),
		matchedPos:   fmt.Sprintf("%s:%d", filepath.Base(patternPos.FilePath), patternPos.LineNumber),
		matchedLevel: pattern.GetLevel(),
		pattern:      pattern,
	}
}

func (bp *BriefPattern) ID() string {
	return bp.id
}

func (bp *BriefPattern) Pattern() *logpattern_go_proto.LogPattern {
	return bp.pattern
}

func (bp *BriefPattern) String() string {
	return fmt.Sprintf("id %s: position %s, level %s, signature %+v", bp.id, bp.matchedPos, bp.matchedLevel, bp.pattern.GetSignature())
}

// PatternMatcher helps to find the log pattern
type PatternMatcher struct {
	store *keyvalue.Store

	trie *patternTrie
}

// MockPatternMatcher mocks a PatternMatcher for testing
func MockPatternMatcher(logs []*logpattern_go_proto.LogPattern) (*PatternMatcher, error) {
	ps := &PatternMatcher{
		trie: NewPatternTrie(),
	}

	for _, lp := range logs {
		if len(lp.Signature) > 0 {
			logSingature := lp.Signature[0]
			err := ps.trie.Insert(logSingature, lp)
			if err != nil {
				return nil, err
			}
		}
	}
	return ps, nil
}

// NewPatternMatcher return a PatternMatcher,
// and the corresponding trie for matching is loaded from the *keyvalue.Store
func NewPatternMatcher(store *keyvalue.Store) (*PatternMatcher, error) {
	ps := &PatternMatcher{
		trie:  NewPatternTrie(),
		store: store,
	}

	err := ps.load(context.Background())
	return ps, err
}

// load reads all log patterns from the store to build a matching trie
// the current algorithm only uses the first log signature to construct the trie
// it can be extended to support multiple sinatures matching in the future
func (p *PatternMatcher) load(ctx context.Context) error {
	return p.store.ScanLogPattern(ctx, func(_, value []byte) error {
		lp := &logpattern_go_proto.LogPattern{}
		err := lp.Unmarshal(value)
		if err != nil {
			return err
		}

		if len(lp.Signature) > 0 {
			logSingature := lp.Signature[0]
			err := p.trie.Insert(logSingature, lp)
			if err != nil {
				return err
			}
		}

		return nil
	})
}

/*
Match return the matched log pattern.
The matching algorithm is as follows
1. Use log sigatures to match log patterns
2. Use log level and position to match log patterns
*/
func (p *PatternMatcher) Match(lp *scanner.Log) *MatchedResult {
	if lp == nil {
		return nil
	}

	return p.trie.Match(lp.Msg, lp.Level, lp.Position)
}
