package cmd

import (
	"errors"
	"fmt"
	"log"     // Import mới
	"syscall" // Import mới

	"github.com/khoahotran/gochain-ledger/application"
	"github.com/spf13/cobra"
	"golang.org/x/term" // Import mới
)

var createWalletCmd = &cobra.Command{
	Use:   "createwallet",
	Short: "Tạo một cặp ví (Keypair) mới (đã mã hóa)",
	Run: func(cmd *cobra.Command, args []string) {

		// 1. Yêu cầu mật khẩu
		fmt.Print("Nhập mật khẩu (để mã hóa ví): ")
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Lỗi khi nhập mật khẩu: %v", err)
		}
		password := string(bytePassword)
		fmt.Println() // Thêm một dòng mới sau khi nhập pass

		if password == "" {
			Handle(errors.New("mật khẩu không được để trống"))
		}

		// 2. Yêu cầu nhập lại mật khẩu
		fmt.Print("Nhập lại mật khẩu: ")
		bytePasswordConfirm, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Lỗi khi nhập mật khẩu: %v", err)
		}
		passwordConfirm := string(bytePasswordConfirm)
		fmt.Println() // Thêm dòng mới

		if password != passwordConfirm {
			Handle(errors.New("mật khẩu không khớp"))
		}

		// 3. Gọi usecase
		application.CreateWalletUseCase(password)
	},
}

func init() {
	rootCmd.AddCommand(createWalletCmd)
}
