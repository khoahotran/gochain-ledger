package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gochain-ledger",
	Short: "GoChain Ledger là một blockchain demo viết bằng Go",
	Run: func(cmd *cobra.Command, args []string) {

		cmd.Help()
	},
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {

}

func Handle(err error) {
	if err != nil {
		fmt.Printf("Lỗi: %s\n", err.Error())
		os.Exit(1)
	}
}
