package cmd

import (
	"bufio"
	"fmt"
	"io"
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
	Template    string
)

func NewAnalyzeCmd() *cobra.Command {
	cmdAnalyze := &cobra.Command{
		Use:   "analyze",
		Short: "Analyze the coverage result and output the analysis result report",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			//Report("/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil/tmp/", OutReport)
			if !Exists(LogCoverage) {
				return fmt.Errorf("log coverage does't not exist")
			}

			var (
				fd  *os.File
				err error
			)
			if !isFile(Output) {
				fd = os.Stdout
			} else {
				fd, err = os.Open(Output)
				if err != nil {
					return err
				}
			}

			writer := bufio.NewWriter(fd)
			writer.Flush()

			Report(LogCoverage, writer)
			return writer.Flush()
		},
	}

	cmdAnalyze.Flags().StringVar(&LogCoverage, "log-coverage", "", "the log coverage directory (contain log coverage, reference code information)")
	cmdAnalyze.Flags().StringVar(&OutReport, "output", "", "output report of log coverage analysis results (default stdout)")
	cmdAnalyze.Flags().StringVar(&Template, "template", "", "output report template, default ")
	cmdAnalyze.MarkFlagRequired("log-coverage")
	return cmdAnalyze
}

func Report(storePath string, output io.Writer) {
	db, err := leveldb.Open(storePath, nil)
	if err != nil {
		log.Fatalf("open leveldb failed %v", err)
	}

	store := keyvalue.NewLogPatternStore(db)

	reporter, err := log_reporter.NewReporter(store, output)
	if err != nil {
		log.Fatalf("create coverage failed %v", err)
	}

	err = reporter.Render(Template, defaultTemplate)
	if err != nil {
		log.Fatalf("output coverage failed %v", err)
	}
}

func isFile(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !s.IsDir()
}

var defaultTemplate = `
total error log {{.Total}}, covered error log {{.Cov}}
{{- println }}
{{- range $path, $cov := .Details -}}
{{if $cov.Coverage }}
path {{$path}} coverrd count {{$cov.Coverage.CovCount}}
log level {{$cov.Pattern.Level}} signatures {{- $cov.Pattern.Signature}}
coverage detail:
{{- range $addr, $count := $cov.Coverage.CovCountByLog}}
file {{$addr}} cover count {{$count}}
{{- end}}
{{- println }}
{{- else}} {{- end}}
{{- end}}
`
