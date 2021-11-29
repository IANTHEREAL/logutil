package matcher

import (
	"context"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	scanner "github.com/IANTHEREAL/logutil/scanner/log_scan"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

type PatternMatcher struct {
	store *keyvalue.Store

	trie *patternTrie
}

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

func NewPatternMatcher(store *keyvalue.Store) (*PatternMatcher, error) {
	ps := &PatternMatcher{
		trie:  NewPatternTrie(),
		store: store,
	}

	err := ps.load(context.Background())
	return ps, err
}

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

func (p *PatternMatcher) Match(lp *scanner.Log) *MatchedResult {
	if lp == nil {
		return nil
	}

	return p.trie.Match(string(lp.Msg), string(lp.Level), string(lp.Position))
}
