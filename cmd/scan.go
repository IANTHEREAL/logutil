package cmd

import (
	"context"
	"fmt"
	"log"
	"strings"

	log_scanner "github.com/IANTHEREAL/logutil/scanner"
	"github.com/IANTHEREAL/logutil/storage/keyvalue"
	"github.com/IANTHEREAL/logutil/storage/leveldb"
	"github.com/spf13/cobra"
)

var (
	LogPattern string
	LogFile    string
)

func NewScanCmd() *cobra.Command {
	cmdScan := &cobra.Command{
		Use:   "scan",
		Short: "Scan program log files and calculate the log coverage result",
		//Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			/*ScanLog("/Users/ianz/Work/go/src/github.com/IANTHEREAL/logutil/tmp/", []string{
				"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-master-0.log",
				"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-master-1.log",
				"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-master-2.log",
				"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-0.log",
				"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-1.log",
				"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-2.log",
				"/Users/ianz/Work/go/src/github.com/pingcap/dm/log/dm-worker-3.log",
			})*/

			if !Exists(LogPattern) {
				return fmt.Errorf("log pattern set doesn't exist")
			}

			files := strings.Split(LogFile, ",")
			if len(files) <= 0 {
				return fmt.Errorf("log files don't exist")
			}

			ScanLog(LogPattern, files)
			return nil
		},
	}

	cmdScan.Flags().StringVar(&LogPattern, "log-pattern", "", "the directory that stores the extracted log pattern information, `log scan` would also add log coverage into it")
	cmdScan.Flags().StringVar(&LogFile, "logs", "", " the program runtime log files, files are separated by commas")
	cmdScan.MarkFlagRequired("log-pattern")
	cmdScan.MarkFlagRequired("logs")
	return cmdScan
}

func ScanLog(storePath string, logs []string) {
	db, err := leveldb.Open(storePath, nil)
	if err != nil {
		log.Fatalf("open leveldb failed %v", err)
	}

	store := keyvalue.NewLogPatternStore(db)

	p, err := log_scanner.NewLogProcessor(store, logs)
	if err != nil {
		log.Fatalf("new scanner pipeline failed %s", err)
	}

	err = p.Run(context.Background())
	if err != nil {
		log.Fatalf("scanner run %s", err)
	}
}
