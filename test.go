package main

import (
	"context"
	"go/build"
	"log"
	"path/filepath"
	"sync"

	"github.com/IANTHEREAL/logutil/extractor/go/analyzer"
	"github.com/IANTHEREAL/logutil/extractor/go/compiler"
	logextractor "github.com/IANTHEREAL/logutil/extractor/go/log"
	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	log_reporter "github.com/IANTHEREAL/logutil/reporter"
	log_scanner "github.com/IANTHEREAL/logutil/scanner"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
)

func main() {
	db, err := leveldb.Open("/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil/tmp/", nil)
	if err != nil {
		log.Fatalf("open leveldb failed %v", err)
	}

	store := keyvalue.NewLogPatternStore(db)

	//testReopoert(store)
	//testMatcher(store)
	testPrintLogPattern(store)
	//testPrintCoverage(store)
	return

	filter := logextractor.NewFilter(nil)
	builder := &logextractor.Builder{}

	path, err := filepath.Abs("./")
	if err != nil {
		log.Fatalf("absolute path %s error %v", path, err)
	}
	repo, err := builder.Build(build.Default, path)
	if err != nil {
		log.Fatalf("build failed %v", err)
	}

	ai := analyzer.NewAstAnalyzer(filter.Filter)
	output := ai.SetupOutput()

	done := sync.WaitGroup{}
	done.Add(1)
	go func() {
		for {
			lp, ok := <-output
			if !ok {
				break
			}

			pattern := lp.(*logpattern_go_proto.LogPattern)
			pattern.Pos.PackagePath = &logpattern_go_proto.PackagePath{
				Repo: repo.GetRepoPath(),
			}

			log.Printf("result log %s", lp)
			err := store.WriteLogPattern(context.Background(), pattern)
			if err != nil {
				log.Printf("wirte log %s failed %v", lp, err)
			}

		}
		done.Done()
	}()

	wg := sync.WaitGroup{}

	err = repo.ForEach(func(pkg *compiler.PackageCompilation) error {
		wg.Add(1)
		go func() {
			pkg.ForEach(func(file *compiler.FileCompilation, helper *analyzer.AstHelper) {
				err := file.RunAnalysis(ai, helper)
				if err != nil {
					log.Fatalf("analysis failed %v", err)
				}
			})
			wg.Done()
		}()

		return nil
	})

	wg.Wait()
	ai.MarkDone()
	if err != nil {
		log.Fatalf("analyze failed %v", err)
	}
	done.Wait()
}

func testPrintLogPattern(store *keyvalue.Store) {
	count := 0
	store.ScanLogPattern(context.Background(), func(_, value []byte) error {
		lp := &logpattern_go_proto.LogPattern{}
		err := lp.Unmarshal(value)
		if err != nil {
			log.Printf("Unmarshal log %s failed %v", lp, err)
		}
		log.Printf("result log %s", lp)
		count++
		return err
	})
	log.Println(count)
}

func testPrintCoverage(store *keyvalue.Store) {
	store.ScanLogCoverage(context.Background(), func(_, value []byte) error {

		lp := &logpattern_go_proto.Coverage{}
		err := lp.Unmarshal(value)
		if err != nil {
			log.Printf("Unmarshal log %s failed %v", lp, err)
		}
		log.Printf("\n\nresult log %s", lp)

		return err
	})
}

func testReopoert(store *keyvalue.Store) {
	reporter, err := log_reporter.NewCoverager(store)
	if err != nil {
		log.Fatalf("create coverage failed %v", err)
	}

	total, cov := reporter.OverallCoverage()
	log.Printf("total error log %d, covered err log %d", total, cov)

	reporter.ForEach(func(l *log_reporter.LogDetail) {
		if l.Coverage != nil {
			log.Printf("%s", l)
		}
	})
}

func testMatcher(store *keyvalue.Store) {
	p, err := log_scanner.NewPipeline(store, "github.com/pingcap/ticdc/dm", []string{
		"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-master-0.log",
		"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-master-1.log",
		"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-master-2.log",
		"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-0.log",
		"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-1.log",
		"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-2.log",
		"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-3.log",
	})
	if err != nil {
		log.Fatalf("new scanner pipeline failed %s", err)
	}

	err = p.Run()
	if err != nil {
		log.Fatalf("scanner run %s", err)
	}

	/*scanner, err := log_scanner.NewLogScanner("/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-master-1.log")
	if err != nil {
		log.Fatalf("create log scanner failed %v", err)
	}

	logs := []*logpattern_go_proto.LogPattern{
		&logpattern_go_proto.LogPattern{
			Pos: &logpattern_go_proto.Position{
				FilePath:   "dm/master/util.go",
				LineNumber: 121,
			},
			Level:     "warn",
			Signature: []string{"\"failed to apply request\""},
		},
		&logpattern_go_proto.LogPattern{
			Pos: &logpattern_go_proto.Position{
				FilePath:   "dm/master/netutil.go",
				LineNumber: 121,
			},
			Level:     "warn",
			Signature: []string{"\"failed to resolve URL Host\""},
		},
	}

	matcher, err := log_scanner.MockPatternMatcher(logs)
	if err != nil {
		log.Fatalf("mock pattern matcher error %v", err)
	}

	coverager := log_scanner.NewCoverager(store)

	lg, err := scanner.Scan()

	for err != io.EOF {
		log.Printf("%s", lg)
		if err != nil {
			log.Fatalf("scan log error %v", err)
		}

		res := matcher.Match(lg)
		if res != nil {
			for _, lp := range res.Patterns {
				log.Printf("macthed %+v", lp)
				coverager.Compute(lg, lp)
			}
		}

		lg, err = scanner.Scan()
	}

	log.Printf("scan error %v", err)

	log.Printf("flush error %s", coverager.Flush())*/
}
