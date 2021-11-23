package main

import (
	"context"
	"go/build"
	"log"

	"github.com/IANTHEREAL/logutil/extractor/go/analyzer"
	"github.com/IANTHEREAL/logutil/extractor/go/compiler"
	logextractor "github.com/IANTHEREAL/logutil/extractor/go/log"
	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
)

func main() {
	db, err := leveldb.Open("/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil/tmp/", nil)
	if err != nil {
		log.Fatalf("open leveldb failed %v", err)
	}

	store := keyvalue.NewLogPatternStore(db)
	store.Scan(context.Background(), func(_, value []byte) error {
		lp := &logpattern_go_proto.LogPattern{}
		err := lp.Unmarshal(value)
		if err != nil {
			log.Printf("Unmarshal log %s failed %v", lp, err)
		}
		log.Printf("\n\nresult log %s", lp)

		return err
	})

	return

	filter := logextractor.NewFilterHub(nil)

	packages, err := logextractor.Build(build.Default, "/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil")
	if err != nil {
		log.Fatalf("build failed %v", err)
	}

	for _, pkg := range packages {
		ai := analyzer.NewAstAnalyzer(filter.Filter)
		output := ai.SetupOutput()
		pkg.ForEach(func(file *compiler.FileCompilation, helper *analyzer.AstHelper) {
			err := file.RunAnalysis(ai, helper)
			if err != nil {
				log.Fatalf("analysis failed %v", err)
			}
		})

		ai.MarkDone()
		for {
			lp, ok := <-output
			if !ok {
				break
			}

			log.Printf("result log %s", lp)
			err := store.Write(context.Background(), lp.(*logpattern_go_proto.LogPattern))
			if err != nil {
				log.Printf("wirte log %s failed %v", lp, err)
			}

			log.Printf("package compliant %+v", pkg.GetPackagePath())
		}
	}
}
