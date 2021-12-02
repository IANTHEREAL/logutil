package cmd

import (
	"context"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/IANTHEREAL/logutil/extractor/go/analyzer"
	"github.com/IANTHEREAL/logutil/extractor/go/compiler"
	logextractor "github.com/IANTHEREAL/logutil/extractor/go/log"
	"github.com/IANTHEREAL/logutil/pkg/util"
	logpattern_go_proto "github.com/IANTHEREAL/logutil/proto"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	"github.com/spf13/cobra"
)

var (
	Codebase    string
	FlterConfig string
	Output      string

	rule *logpattern_go_proto.LogPatternRule
)

func NewExtractCmd() *cobra.Command {
	cmdExtract := &cobra.Command{
		Use:          "extract",
		Short:        "Extract logs pattern and reference code information from codebase and compilation",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// handle filter config
			if FlterConfig != "" {
				rule = &logpattern_go_proto.LogPatternRule{}
				// read from config file
				if err := util.StrictDecodeFile(FlterConfig, rule); err != nil {
					return err
				}
			} else {
				// set default config, log_level = ["error"]
				rule = &logpattern_go_proto.LogPatternRule{
					LogLevel: []string{"error"},
				}
			}

			if !Exists(Codebase) {
				return fmt.Errorf("code %s doesn't exists", Codebase)
			}

			if Output == "" {
				absOutput, err := filepath.Abs(Codebase)
				if err != nil {
					return err
				}

				Output = fmt.Sprintf("./%s.logpattern", filepath.Base(absOutput))
			}

			db, err := leveldb.Open(Output, nil)
			if err != nil {
				log.Fatalf("open leveldb failed %v", err)
			}
			store := keyvalue.NewLogPatternStore(db)

			ExtractLogPattern(store, Codebase, rule)
			return nil
		},
	}

	cmdExtract.Flags().StringVar(&Codebase, "codebase", "./", "Source codebase directory for extracting log information")
	cmdExtract.Flags().StringVar(&FlterConfig, "filter", "", "the log filter rule config file using json format, if no config file, default set logLevel = error")
	cmdExtract.Flags().StringVar(&Output, "output", "", "the output file that stores the extracted log pattern and reference code information(default \"./${codebase-dirname}.logpattern\")")
	return cmdExtract
}

func ExtractLogPattern(store *keyvalue.Store, codebase string, rule *logpattern_go_proto.LogPatternRule) {
	err := store.WriteLogPatternRule(context.Background(), rule)
	if err != nil {
		log.Fatalf("save log pattern rule into log patern store failed %v", err)
	}

	filter := logextractor.NewFilter(rule)
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

func Exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}
