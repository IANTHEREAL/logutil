package log_scanner

import (
	"context"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
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

func NewPatternMatcher(repo string, store *keyvalue.Store) (*PatternMatcher, error) {
	ps := &PatternMatcher{
		trie:  NewPatternTrie(),
		store: store,
	}

	err := ps.load(context.Background(), repo)
	return ps, err
}

func (p *PatternMatcher) load(ctx context.Context, repo string) error {
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

func (p *PatternMatcher) Match(lp *Log) *MatchedResult {
	if lp == nil {
		return nil
	}

	/*if strings.ToLower(string(lp.Level)) == "error" {

		log.Printf("log %s", lp)
	}*/

	return p.trie.Match(string(lp.Msg), string(lp.Level), string(lp.Position))
}
