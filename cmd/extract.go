package cmd

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
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	"github.com/spf13/cobra"
)

var (
	Codebase    string
	FlterConfig string
	Output      string
)

func NewExtractCmd() *cobra.Command {
	cmdExtract := &cobra.Command{
		Use:          "extract",
		Short:        "Extract logs pattern and reference code information from codebase and compilation",
		SilenceUsage: true,
		Run: func(cmd *cobra.Command, args []string) {
			ExtractLogPattern(Codebase, nil, "/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil/tmp/")
		},
	}

	cmdExtract.Flags().StringVar(&Codebase, "codebase", "./", "Source codebase directory for extracting log information")
	cmdExtract.Flags().StringVar(&FlterConfig, "filter", "", "the log filter rule config file, using log level `log-level:[Error|Fatal]` or log keywords `keywords:[regex1,regex2]`")
	cmdExtract.Flags().StringVar(&Output, "output", "", "the output file that stores the extracted log pattern and reference code information(default \"./${codebase-dirname}.logpattern\")")
	return cmdExtract
}

func ExtractLogPattern(codebase string, rule *logextractor.LogPatternRule, storePath string) {
	db, err := leveldb.Open(storePath, nil)
	if err != nil {
		log.Fatalf("open leveldb failed %v", err)
	}

	store := keyvalue.NewLogPatternStore(db)

	filter := logextractor.NewFilter(nil)
	builder := &logextractor.Builder{}

	path, err := filepath.Abs(codebase)
	if err != nil {
		log.Fatalf("absolute path %s error %v", codebase, err)
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

			//log.Printf("result log %s", lp)
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
		//go func() {
		pkg.ForEach(func(file *compiler.FileCompilation, helper *analyzer.AstHelper) {
			err := file.RunAnalysis(ai, helper)
			if err != nil {
				log.Fatalf("analysis failed %v", err)
			}
		})
		wg.Done()
		//}()

		return nil
	})

	wg.Wait()
	ai.MarkDone()
	if err != nil {
		log.Fatalf("analyze failed %d", err)
	}
	done.Wait()
}