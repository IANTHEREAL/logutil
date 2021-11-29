package recorder

import (
	"fmt"
	"log"
	"sync"

	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	scanner "github.com/IANTHEREAL/logutil/scanner/log_scan"
)

// UnknowLogRecord used to record the log that exits in log files
// but not captured by log extractor
// Now support printing unknow logs through Flush method
type UnknowLogRecord struct {
	sync.RWMutex
	logs map[string]*logpattern_go_proto.UnknowLogPattern
}

func NewUnknowLogRecorder() *UnknowLogRecord {
	return &UnknowLogRecord{
		logs: make(map[string]*logpattern_go_proto.UnknowLogPattern),
	}
}

func (r *UnknowLogRecord) Record(l *scanner.Log) {
	r.Lock()
	log := r.logs[l.Position]
	if log == nil {
		log = &logpattern_go_proto.UnknowLogPattern{
			Pos: &logpattern_go_proto.Position{
				FilePath: l.Position,
			},
			Level:         l.Level,
			CovCountByLog: make(map[string]int32),
		}
		r.logs[l.Position] = log
	}

	log.CovCount = log.CovCount + 1
	if count, ok := log.CovCountByLog[l.LogPath]; !ok {
		log.CovCountByLog[l.LogPath] = 1
	} else {
		log.CovCountByLog[l.LogPath] = count + 1
	}
	r.Unlock()
}

func (r *UnknowLogRecord) Flush() error {
	r.Lock()
	defer r.Unlock()

	if len(r.logs) == 0 {
		return nil
	}

	result := fmt.Sprintf("total %d unknow logs.  exmaples:", len(r.logs))
	index := 0
	for _, log := range r.logs {
		index++
		result += fmt.Sprintf("[%s] [%s] [count=%d] [map %+v];", log.Pos, log.Level, log.CovCount, log.CovCountByLog)
		if index >= 32 {
			break
		}
	}

	log.Printf("unknow logs  %s ...", result)
	return nil
}
