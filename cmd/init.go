package cmd

import (
	"errors"

	"github.com/khoahotran/gochain-ledger/application"
	"github.com/spf13/cobra"
)

var initChainCmd = &cobra.Command{
	Use:   "init",
	Short: "Khởi tạo blockchain mới và đào block Genesis",
	Run: func(cmd *cobra.Command, args []string) {
		address, _ := cmd.Flags().GetString("address")
		if address == "" {
			Handle(errors.New("Cần cung cấp địa chỉ ví nhận thưởng (flag --address)"))
		}
		application.InitChainUseCase(address)
	},
}

func init() {
	initChainCmd.Flags().String("address", "", "Địa chỉ ví nhận thưởng")
	rootCmd.AddCommand(initChainCmd)
}
