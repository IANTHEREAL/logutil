package cmd

import (
	"log"
	"os"

	log_reporter "github.com/IANTHEREAL/logutil/reporter"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	"github.com/spf13/cobra"
)

var (
	LogCoverage string
	OutReport   string
)

func NewAnalyzeCmd() *cobra.Command {
	cmdAnalyze := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze the coverage result and output the analysis result report",
		//Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			Report("/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil/tmp/", OutReport)
		},
	}

	cmdAnalyze.Flags().StringVar(&LogCoverage, "log-coverage", "", "the log coverage directory (contain log coverage, reference code information)")
	cmdAnalyze.Flags().StringVar(&OutReport, "output", "", "output report of log coverage analysis results (default stdout)")
	cmdAnalyze.MarkFlagRequired("log-coverage")
	return cmdAnalyze
}

func Report(storePath string, output string) {
	db, err := leveldb.Open(storePath, nil)
	if err != nil {
		log.Fatalf("open leveldb failed %v", err)
	}

	store := keyvalue.NewLogPatternStore(db)

	reporter, err := log_reporter.NewReporter(store, os.Stdout, "/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil/reporter/report_test_template.md")
	if err != nil {
		log.Fatalf("create coverage failed %v", err)
	}

	err = reporter.Output()
	if err != nil {
		log.Fatalf("output coverage failed %v", err)
	}
}
