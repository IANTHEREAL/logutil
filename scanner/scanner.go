package log_scanner

import (
	"context"
	"io"
	"log"
	"sync"

	matcher "github.com/IANTHEREAL/logutil/scanner/log_match"
	recorder "github.com/IANTHEREAL/logutil/scanner/log_record"
	scanner "github.com/IANTHEREAL/logutil/scanner/log_scan"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

// LogProcessor is responsible for scanning the log,
// looking for the log matching the log pattern,
// and then recording the relevant coverage data
type LogProcessor struct {
	store *keyvalue.Store

	matcher    *matcher.PatternMatcher
	scannerSet []*scanner.LogScanner
	coverager  *recorder.Coverager

	unknowLogs *recorder.UnknowLogRecord
}

func NewLogProcessor(store *keyvalue.Store, logPaths []string) (*LogProcessor, error) {
	matcher, err := matcher.NewPatternMatcher(store)
	if err != nil {
		return nil, err
	}

	coverager := recorder.NewCoverager(store)
	unknowLogs := recorder.NewUnknowLogRecorder()

	scannerSet := make([]*scanner.LogScanner, 0, len(logPaths))
	for _, path := range logPaths {
		scanner, err := scanner.NewLogScanner(path)
		if err != nil {
			return nil, err
		}
		scannerSet = append(scannerSet, scanner)
	}

	return &LogProcessor{
		scannerSet: scannerSet,
		coverager:  coverager,
		matcher:    matcher,
		unknowLogs: unknowLogs,
	}, nil
}

// Run starts the log processing
func (s *LogProcessor) Run(ctx context.Context) error {
	//pipeline := NewPipiline(1024)
	wg := sync.WaitGroup{}

	for _, scanner := range s.scannerSet {
		line := newAssemLine(s.matcher, scanner, s.coverager, s.unknowLogs)
		wg.Add(1)
		go func(l *assemLine) {
			if err := l.run(ctx); err != nil {
				log.Printf("process line %s meet error %v", l.scanner.GetLogPath(), err)
			}
			wg.Done()
		}(line)
	}

	wg.Wait()
	s.unknowLogs.Flush()
	return s.coverager.Flush()
}

// assemLine is a separate logics processing flow,
// including log scan, log matching, and coverage recording process unit
type assemLine struct {
	matcher    *matcher.PatternMatcher
	scanner    *scanner.LogScanner
	coverager  *recorder.Coverager
	unknowLogs *recorder.UnknowLogRecord
}

func newAssemLine(matcher *matcher.PatternMatcher, scanner *scanner.LogScanner, coverager *recorder.Coverager, unknowLogs *recorder.UnknowLogRecord) *assemLine {
	return &assemLine{
		matcher:    matcher,
		scanner:    scanner,
		coverager:  coverager,
		unknowLogs: unknowLogs,
	}
}

// run starts and runs this processing logic line
func (l *assemLine) run(ctx context.Context) error {
	pipeline := NewPipiline(1024)
	wg := sync.WaitGroup{}

	lineCtx, cancel := context.WithCancel(ctx)
	var (
		err1   error
		err2   error
		retErr error
	)

	wg.Add(1)
	go func() {
		err1 = l.runLogScan(lineCtx, pipeline)
		if err1 != nil {
			log.Printf("scan log meets error %v", err1)
		}
		retErr = err1
		pipeline.Close()
		wg.Done()
	}()

	err2 = l.runMathAndRecord(ctx, pipeline)
	cancel()
	wg.Wait()

	if err2 != nil && err2 != context.DeadlineExceeded {
		log.Printf("match log meets error %v", err2)
		retErr = err2
	}

	return retErr
}

// runLogScan scans log and send log to pipeline
func (l *assemLine) runLogScan(ctx context.Context, pipeline *Pipeline) error {
	for {
		lg, err := l.scanner.Scan()
		if err == io.EOF {
			return nil
		} else if err != nil {
			// TODO: implement it as design document described
			return err
		}

		err = pipeline.Write(ctx, &Payload{log: lg})
		if err != nil {
			return err
		}
	}
}

// runMathAndRecord recieve log from pipeline and match them with log pattern，then record the coverage
func (l *assemLine) runMathAndRecord(ctx context.Context, pipeline *Pipeline) error {
	for {
		payload, err := pipeline.Read(ctx)
		if err == io.EOF {
			return nil
		} else if err != nil {
			return err
		}

		res := l.matcher.Match(payload.log)
		// TODO: check unknon log pattern need to config log level
		if (res == nil || len(res.Patterns) == 0) && payload.log.Level == "error" {
			l.unknowLogs.Record(payload.log)
		} else {
			for _, lp := range res.Patterns {
				l.coverager.Record(payload.log, lp)
			}
		}
	}
}
