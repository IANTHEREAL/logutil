package log_reporter

import (
	"context"
	"fmt"
	"log"

	"github.com/IANTHEREAL/logutil/pkg/util"
	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

type LogDetail struct {
	Pattern  *logpattern_go_proto.LogPattern
	Coverage *logpattern_go_proto.Coverage
}

func (l *LogDetail) String() string {
	format := "repo %s log position %s:%d:%d cover count %d\n log level %s signatures %v\n "
	covercount := 0
	if l.Coverage != nil {
		covercount = int(l.Coverage.CovCount)
		if len(l.Coverage.CovCountByLog) > 0 {
			format = format + "coverage detail:\n"
			for addr, cov := range l.Coverage.CovCountByLog {
				format = format + fmt.Sprintf("file %s cover count %d\n", addr, cov)
			}
		}
	}
	return fmt.Sprintf(format, l.Pattern.Pos.PackagePath.Repo, l.Pattern.Pos.FilePath, l.Pattern.Pos.LineNumber, l.Pattern.Pos.ColumnOffset, covercount, l.Pattern.Level, l.Pattern.Signature)
}

type Coverager struct {
	Details map[string]*LogDetail

	Total, Cov int

	store *keyvalue.Store
}

func NewCoverager(store *keyvalue.Store) (*Coverager, error) {
	cov := &Coverager{
		store:   store,
		Details: make(map[string]*LogDetail),
	}

	err := cov.load(context.Background())
	return cov, err
}

func (c *Coverager) OverallCoverage() (int, int) {
	return c.Total, c.Cov
}

func (c *Coverager) load(ctx context.Context) error {
	err := c.store.ScanLogPattern(ctx, func(_, value []byte) error {
		lp := &logpattern_go_proto.LogPattern{}
		err := lp.Unmarshal(value)
		if err != nil {
			return err
		}

		path := util.PosToStr(lp.Pos)
		if d := c.Details[path]; d == nil {
			c.Total++
			c.Details[path] = &LogDetail{
				Pattern: lp,
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	return c.store.ScanLogCoverage(ctx, func(_, value []byte) error {
		lp := &logpattern_go_proto.Coverage{}
		err := lp.Unmarshal(value)
		if err != nil {
			return err
		}

		path := util.PosToStr(lp.Pos)
		if d := c.Details[path]; d != nil {
			c.Cov++
			d.Coverage = lp
		} else {
			log.Fatalf("not found reference log %s", lp)
		}

		return nil
	})
}
