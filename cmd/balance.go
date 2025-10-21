package cmd

import (
	"context" // Import mới
	"errors"
	"fmt"
	"log" // Import mới

	// "github.com/khoahotran/gochain-ledger/application" // XÓA BỎ
	"github.com/khoahotran/gochain-ledger/proto" // Import mới
	"github.com/spf13/cobra"
	"google.golang.org/grpc"                      // Import mới
	"google.golang.org/grpc/credentials/insecure" // Import mới
)

var getBalanceCmd = &cobra.Command{
	Use:   "balance",
	Short: "Kiểm tra số dư của một địa chỉ ví (qua một node)",
	Run: func(cmd *cobra.Command, args []string) {
		address, _ := cmd.Flags().GetString("address")
		nodeAddr, _ := cmd.Flags().GetString("node") // Lấy flag node

		if address == "" || nodeAddr == "" {
			Handle(errors.New("Cần cung cấp flag --address và --node"))
		}

		// --- Logic gRPC Client ---

		// 1. Kết nối đến node
		conn, err := grpc.Dial(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Không thể kết nối: %v", err)
		}
		defer conn.Close()

		// 2. Tạo client service
		client := proto.NewNodeServiceClient(conn)

		// 3. Chuẩn bị request
		req := &proto.GetBalanceRequest{Address: address}

		// 4. Gọi hàm gRPC
		res, err := client.GetBalance(context.Background(), req)
		if err != nil {
			log.Fatalf("Gọi gRPC GetBalance thất bại: %v", err)
		}

		// 5. In kết quả
		fmt.Printf("Số dư của '%s': %d\n", address, res.Balance)
	},
}

func init() {
	getBalanceCmd.Flags().String("address", "", "Địa chỉ ví cần kiểm tra")
	// Thêm flag --node
	getBalanceCmd.Flags().String("node", "localhost:50051", "Địa chỉ node đang chạy")
	rootCmd.AddCommand(getBalanceCmd)
}
