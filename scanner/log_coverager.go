package log_scanner

import (
	"context"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

type Coverager struct {
	logCoverageCount map[*logpattern_go_proto.Position]*logpattern_go_proto.Coverage

	store *keyvalue.Store
}

func NewCoverager(store *keyvalue.Store) *Coverager {
	return &Coverager{
		store:            store,
		logCoverageCount: make(map[*logpattern_go_proto.Position]*logpattern_go_proto.Coverage),
	}
}

func (c *Coverager) Compute(l *Log, pattern *logpattern_go_proto.LogPattern) {
	cov := c.logCoverageCount[pattern.Pos]
	if cov == nil {
		cov = &logpattern_go_proto.Coverage{
			Pos:           pattern.Pos,
			CovCountByLog: make(map[string]int32),
		}
		c.logCoverageCount[pattern.Pos] = cov
	}

	cov.CovCount = cov.CovCount + 1
	if count, ok := cov.CovCountByLog[l.logPath]; !ok {
		cov.CovCountByLog[l.logPath] = 1
	} else {
		cov.CovCountByLog[l.logPath] = count + 1
	}
}

func (c *Coverager) Flush() error {
	for _, cov := range c.logCoverageCount {
		err := c.store.WriteLogCoverage(context.Background(), cov)
		if err != nil {
			return err
		}
	}

	return nil
}
