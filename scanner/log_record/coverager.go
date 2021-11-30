package recorder

import (
	"context"
	"sync"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	matcher "github.com/IANTHEREAL/logutil/scanner/log_match"
	scanner "github.com/IANTHEREAL/logutil/scanner/log_scan"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

type Coverager struct {
	sync.RWMutex
	logCoverageCount map[string]*logpattern_go_proto.Coverage

	store *keyvalue.Store
}

func NewCoverager(store *keyvalue.Store) *Coverager {
	return &Coverager{
		store:            store,
		logCoverageCount: make(map[string]*logpattern_go_proto.Coverage),
	}
}

func (c *Coverager) Record(l *scanner.Log, pattern *matcher.BriefPattern) {
	c.Lock()
	cov := c.logCoverageCount[pattern.ID()]
	if cov == nil {
		cov = &logpattern_go_proto.Coverage{
			Pos:           pattern.Pattern().GetPos(),
			CovCountByLog: make(map[string]int32),
		}
		c.logCoverageCount[pattern.ID()] = cov
	}

	cov.CovCount = cov.CovCount + 1
	if count, ok := cov.CovCountByLog[l.LogPath]; !ok {
		cov.CovCountByLog[l.LogPath] = 1
	} else {
		cov.CovCountByLog[l.LogPath] = count + 1
	}
	c.Unlock()
}

func (c *Coverager) Flush() error {
	c.Lock()
	defer c.Unlock()

	for _, cov := range c.logCoverageCount {
		err := c.store.WriteLogCoverage(context.Background(), cov)
		if err != nil {
			return err
		}
	}

	return nil
}
