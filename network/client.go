package network

import (
	"context"
	"log"

	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func SendTransactionToNode(targetNodeAddr string, tx *domain.Transaction) {
	log.Printf("Đang kết nối đến node tại %s...", targetNodeAddr)

	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối: %v", err)
	}
	defer conn.Close()

	client := proto.NewNodeServiceClient(conn)

	protoTx := MapDomainTransactionToProto(tx)

	ack, err := client.SendTransaction(context.Background(), protoTx)
	if err != nil {
		log.Fatalf("Gọi gRPC SendTransaction thất bại: %v", err)
	}

	log.Printf("Phản hồi từ node: [%t] %s", ack.Success, ack.Message)
}
