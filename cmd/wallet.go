package cmd

import (
	"errors"
	"fmt"
	"log"
	"syscall"

	"github.com/khoahotran/gochain-ledger/application"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var createWalletCmd = &cobra.Command{
	Use:   "createwallet",
	Short: "Tạo một cặp ví (Keypair) mới (đã mã hóa)",
	Run: func(cmd *cobra.Command, args []string) {

		fmt.Print("Nhập mật khẩu (để mã hóa ví): ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Lỗi khi nhập mật khẩu: %v", err)
		}
		password := string(bytePassword)
		fmt.Println()

		if password == "" {
			Handle(errors.New("mật khẩu không được để trống"))
		}

		fmt.Print("Nhập lại mật khẩu: ")
		bytePasswordConfirm, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Lỗi khi nhập mật khẩu: %v", err)
		}
		passwordConfirm := string(bytePasswordConfirm)
		fmt.Println()

		if password != passwordConfirm {
			Handle(errors.New("mật khẩu không khớp"))
		}

		application.CreateWalletUseCase(password)
	},
}

func init() {
	rootCmd.AddCommand(createWalletCmd)
}
