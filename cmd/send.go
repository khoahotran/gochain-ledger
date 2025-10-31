package cmd

import (
	"errors"
	"fmt"
	"log"
	"syscall"

	"github.com/khoahotran/gochain-ledger/application"
	"github.com/khoahotran/gochain-ledger/wallet"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var sendCmd = &cobra.Command{
	Use:   "send",
	Short: "Gửi tiền từ ví A đến ví B",
	Run: func(cmd *cobra.Command, args []string) {
		from, _ := cmd.Flags().GetString("from")
		to, _ := cmd.Flags().GetString("to")
		amount, _ := cmd.Flags().GetInt64("amount")
		nodeAddr, _ := cmd.Flags().GetString("node")

		if from == "" || to == "" || amount <= 0 || nodeAddr == "" {
			Handle(errors.New("Flag --from, --to, --amount, --key, --node là bắt buộc"))
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

		application.SendUseCase(from, to, amount, loadedWallet, nodeAddr)
	},
}

func init() {
	sendCmd.Flags().String("from", "", "Địa chỉ ví gửi (tên file wallet)")
	sendCmd.Flags().String("to", "", "Địa chỉ ví nhận")
	sendCmd.Flags().Int64("amount", 0, "Số tiền")
	sendCmd.Flags().String("node", "localhost:50051", "Địa chỉ node đang chạy")

	rootCmd.AddCommand(sendCmd)
}
