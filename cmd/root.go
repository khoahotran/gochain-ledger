package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Đây là lệnh gốc, ví dụ: "gochain-ledger"
var rootCmd = &cobra.Command{
	Use:   "gochain-ledger",
	Short: "GoChain Ledger là một blockchain demo viết bằng Go",
	Run: func(cmd *cobra.Command, args []string) {
		// Nếu chạy không, chỉ in trợ giúp
		cmd.Help()
	},
}

// Execute là hàm được gọi bởi main.go
func Execute() error {
	return rootCmd.Execute()
}

// init() được Go gọi tự động
func init() {
	// Ở đây chúng ta sẽ thêm các lệnh con
	// Ví dụ:
	// rootCmd.AddCommand(initChainCmd)
	// rootCmd.AddCommand(createWalletCmd)
	// rootCmd.AddCommand(getBalanceCmd)
	// rootCmd.AddCommand(sendCmd)
	// rootCmd.AddCommand(printChainCmd)
}

// Helper function để xử lý lỗi
func Handle(err error) {
	if err != nil {
		fmt.Printf("Lỗi: %s\n", err.Error())
		os.Exit(1)
	}
}
