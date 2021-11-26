package main

import (
	"github.com/IANTHEREAL/logutil/cmd"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "logcov",
		Short: "logcov is a tool that computes the coverage of exception error handling by analyzing the testing log",
	}
	rootCmd.AddCommand(cmd.NewExtractCmd(), cmd.NewScanCmd(), cmd.NewAnalyzeCmd())
	rootCmd.Execute()
}
