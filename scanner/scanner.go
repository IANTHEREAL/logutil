package log_scanner

import (
	"fmt"
	"io"
	"log"

	"github.com/IANTHEREAL/logutil/storage/keyvalue"
)

type Pipeline struct {
	store *keyvalue.Store

	matcher    *PatternMatcher
	scannerSet []*LogScanner
	coverager  *Coverager
}

func NewPipeline(store *keyvalue.Store, repo string, logPaths []string) (*Pipeline, error) {
	matcher, err := NewPatternMatcher(repo, store)
	if err != nil {
		return nil, err
	}

	coverager := NewCoverager(store)
	scannerSet := make([]*LogScanner, 0, len(logPaths))
	for _, path := range logPaths {
		scanner, err := NewLogScanner(path)
		if err != nil {
			return nil, err
		}
		scannerSet = append(scannerSet, scanner)
	}

	return &Pipeline{
		scannerSet: scannerSet,
		coverager:  coverager,
		matcher:    matcher,
	}, nil
}

func (p *Pipeline) Run() error {
	count := 0
	ak := 0
	for _, scanner := range p.scannerSet {
		lg, err := scanner.Scan()
		for err != io.EOF {
			if err != nil {
				return fmt.Errorf("scan log error %v", err)
			}

			if string(lg.Level) == "ERROR" {
				count++
				//	log.Printf("signatures %s", line)
			}

			res := p.matcher.Match(lg)
			if res != nil {
				for _, lp := range res.Patterns {
					//log.Printf("macthed %+v", lp)
					p.coverager.Compute(lg, lp)
					ak++
				}

				if len(res.Patterns) == 0 {
					if string(lg.Level) == "ERROR" {
						log.Printf("unmacthed %+v", lg)
					}
				}

			} else if string(lg.Level) == "ERROR" {
				log.Printf("unmacthed %+v", lg)
			}

			lg, err = scanner.Scan()
		}
	}

	log.Printf("#####signatures %d %d", count, ak)

	return p.coverager.Flush()
}
