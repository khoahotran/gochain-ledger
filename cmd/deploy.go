package cmd

import (
	"errors"
	"fmt"
	"log"
	"os"
	"syscall"

	"github.com/khoahotran/gochain-ledger/application"
	"github.com/khoahotran/gochain-ledger/wallet"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Triển khai một Smart Contract lên blockchain",
	Run: func(cmd *cobra.Command, args []string) {
		from, _ := cmd.Flags().GetString("from")
		filePath, _ := cmd.Flags().GetString("file")
		nodeAddr, _ := cmd.Flags().GetString("node")

		if from == "" || filePath == "" || nodeAddr == "" {
			Handle(errors.New("Flag --from, --file, --node là bắt buộc"))
		}

		code, err := os.ReadFile(filePath)
		if err != nil {
			Handle(fmt.Errorf("không đọc được file: %v", err))
		}

		fmt.Printf("Nhập mật khẩu cho ví '%s': ", from)
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Lỗi khi nhập mật khẩu: %v", err)
		}
		password := string(bytePassword)
		fmt.Println()

		loadedWallet, err := wallet.LoadAndDecrypt(from, password)
		if err != nil {
			Handle(err)
		}

		application.DeployContractUseCase(from, code, loadedWallet, nodeAddr)
	},
}

func init() {
	deployCmd.Flags().String("from", "", "Địa chỉ ví gửi (chủ sở hữu)")
	deployCmd.Flags().String("file", "", "Đường dẫn đến file .lua của contract")
	deployCmd.Flags().String("node", "localhost:50051", "Địa chỉ node đang chạy")
	rootCmd.AddCommand(deployCmd)
}
