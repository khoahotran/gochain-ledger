package network

import (
	"context"
	"log"

	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/proto" // Import proto
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure" // Dùng để kết nối không cần SSL
)

// SendTransactionToNode kết nối đến một node và gọi RPC SendTransaction
func SendTransactionToNode(targetNodeAddr string, tx *domain.Transaction) {
	log.Printf("Đang kết nối đến node tại %s...", targetNodeAddr)

	// 1. Tạo kết nối đến server
	// Tạm thời dùng WithInsecure() vì chúng ta không có SSL
	conn, err := grpc.Dial(targetNodeAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Không thể kết nối: %v", err)
	}
	defer conn.Close()

	// 2. Tạo client service
	client := proto.NewNodeServiceClient(conn)

	// 3. Map domain transaction sang proto transaction
	protoTx := MapDomainTransactionToProto(tx)

	// 4. Gọi hàm gRPC
	ack, err := client.SendTransaction(context.Background(), protoTx)
	if err != nil {
		log.Fatalf("Gọi gRPC SendTransaction thất bại: %v", err)
	}

	// 5. In kết quả
	log.Printf("Phản hồi từ node: [%t] %s", ack.Success, ack.Message)
}

// (Tương lai: Thêm các hàm client khác ở đây, ví dụ: AnnounceBlockToNode)
