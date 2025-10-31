package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/khoahotran/gochain-ledger/proto"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var getBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Kiểm tra số dư của một địa chỉ ví (qua một node)",
	Run: func(cmd *cobra.Command, args []string) {
		address, _ := cmd.Flags().GetString("address")
		nodeAddr, _ := cmd.Flags().GetString("node")

		if address == "" || nodeAddr == "" {
			Handle(errors.New("Cần cung cấp flag --address và --node"))
		}

		conn, err := grpc.Dial(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Không thể kết nối: %v", err)
		}
		defer conn.Close()

		client := proto.NewNodeServiceClient(conn)

		req := &proto.GetBalanceRequest{Address: address}

		res, err := client.GetBalance(context.Background(), req)
		if err != nil {
			log.Fatalf("Gọi gRPC GetBalance thất bại: %v", err)
		}

		fmt.Printf("Số dư của '%s': %d\n", address, res.Balance)
	},
}

func init() {
	getBalanceCmd.Flags().String("address", "", "Địa chỉ ví cần kiểm tra")

	getBalanceCmd.Flags().String("node", "localhost:50051", "Địa chỉ node đang chạy")
	rootCmd.AddCommand(getBalanceCmd)
}
