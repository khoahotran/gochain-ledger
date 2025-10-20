package cmd

import (
	"encoding/json" // IMPORT MỚI
	"errors"
	"fmt"
	"log"
	"syscall"

	"github.com/khoahotran/gochain-ledger/application"
	"github.com/khoahotran/gochain-ledger/wallet"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var callCmd = &cobra.Command{
	Use:   "call",
	Short: "Gọi một hàm trên Smart Contract đã triển khai",
	Run: func(cmd *cobra.Command, args []string) {
		from, _ := cmd.Flags().GetString("from")
		contractAddr, _ := cmd.Flags().GetString("contract")
		funcName, _ := cmd.Flags().GetString("function")
		jsonArgs, _ := cmd.Flags().GetString("args")
		nodeAddr, _ := cmd.Flags().GetString("node")

		if from == "" || contractAddr == "" || funcName == "" || nodeAddr == "" {
			Handle(errors.New("Flag --from, --contract, --function, --node là bắt buộc"))
		}

		// 1. Giải mã (parse) chuỗi JSON args
		var parsedArgs []interface{}
		if jsonArgs != "" {
			err := json.Unmarshal([]byte(jsonArgs), &parsedArgs)
			if err != nil {
				Handle(fmt.Errorf("lỗi parse --args (phải là JSON array): %v", err))
			}
		}

		// 2. Yêu cầu mật khẩu
		fmt.Printf("Nhập mật khẩu cho ví '%s': ", from)
		bytePassword, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			log.Fatalf("Lỗi khi nhập mật khẩu: %v", err)
		}
		password := string(bytePassword)
		fmt.Println()

		// 3. Tải ví
		loadedWallet, err := wallet.LoadAndDecrypt(from, password)
		if err != nil {
			Handle(err)
		}

		// 4. Gọi Use Case
		application.CallContractUseCase(from, contractAddr, funcName, parsedArgs, loadedWallet, nodeAddr)
	},
}

func init() {
	callCmd.Flags().String("from", "", "Địa chỉ ví gửi (người gọi)")
	callCmd.Flags().String("contract", "", "Địa chỉ Contract (ID của TX deploy)")
	callCmd.Flags().String("function", "", "Tên hàm Lua để gọi")
	callCmd.Flags().String("args", "[]", "Các tham số (dạng JSON array, ví dụ: '[\"hello\", 123]')")
	callCmd.Flags().String("node", "localhost:3000", "Địa chỉ node đang chạy")
	rootCmd.AddCommand(callCmd)
}
