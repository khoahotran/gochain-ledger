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

var readCmd = &cobra.Command{
	Use:   "read",
	Short: "Đọc một key từ state của Smart Contract (chỉ đọc)",
	Run: func(cmd *cobra.Command, args []string) {
		contractAddr, _ := cmd.Flags().GetString("contract")
		key, _ := cmd.Flags().GetString("key")
		nodeAddr, _ := cmd.Flags().GetString("node")

		if contractAddr == "" || key == "" || nodeAddr == "" {
			Handle(errors.New("Flag --contract, --key, --node là bắt buộc"))
		}

		conn, err := grpc.Dial(nodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Fatalf("Không thể kết nối: %v", err)
		}
		defer conn.Close()

		client := proto.NewNodeServiceClient(conn)
		req := &proto.GetContractStateRequest{
			ContractAddress: contractAddr,
			Key:             key,
		}

		res, err := client.GetContractState(context.Background(), req)
		if err != nil {
			log.Fatalf("Gọi gRPC GetContractState thất bại: %v", err)
		}

		fmt.Printf("State[%s]: '%s'\n", key, res.Value)
	},
}

func init() {
	readCmd.Flags().String("contract", "", "Địa chỉ Contract (ID của TX deploy)")
	readCmd.Flags().String("key", "", "Tên key cần đọc")
	readCmd.Flags().String("node", "localhost:50051", "Địa chỉ node đang chạy")
	rootCmd.AddCommand(readCmd)
}
