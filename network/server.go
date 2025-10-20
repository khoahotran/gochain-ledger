package network

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"net"

	"github.com/go-redis/redis/v8" // Import Redis
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/proto" // Import proto
	"google.golang.org/grpc"
)

// Server struct sẽ chứa logic
type Server struct {
	proto.UnimplementedNodeServiceServer // Bắt buộc phải có
	Blockchain                           *domain.Blockchain
	RedisClient                          *redis.Client // Mempool
}

const (
	mempoolKey = "gochain:mempool" // Key cho Redis
)

// StartServer khởi động gRPC server
func StartServer(port string, bc *domain.Blockchain, rdb *redis.Client) { // 1. Khởi động Redis client

	// 2. Tạo listener
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Không thể lắng nghe trên cổng %s: %v", port, err)
	}

	// 3. Tạo gRPC server mới
	s := grpc.NewServer()

	// 4. Đăng ký service của chúng ta
	// Truyền blockchain và redis client vào struct Server
	proto.RegisterNodeServiceServer(s, &Server{
		Blockchain:  bc,
		RedisClient: rdb,
	})

	// 5. Bắt đầu phục vụ
	log.Printf("gRPC Server đang lắng nghe tại %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC Server thất bại: %v", err)
	}
}

// === Triển khai các hàm gRPC (Logic mạng) ===

// SendTransaction (Khi 1 node khác gửi transaction đến)
func (s *Server) SendTransaction(ctx context.Context, req *proto.Transaction) (*proto.Ack, error) {
	log.Printf("Nhận được giao dịch mới: %x", req.Id)

	tx := MapProtoTransactionToDomain(req)

	// TODO: Xác thực giao dịch (Verify)
	// (Tạm thời chúng ta tin tưởng)

	// Thêm vào Mempool (Redis)
	// Serialize giao dịch để lưu vào Redis
	var txData bytes.Buffer
	enc := gob.NewEncoder(&txData)
	err := enc.Encode(tx)
	if err != nil {
		log.Printf("Lỗi serialize tx: %v", err)
		return &proto.Ack{Success: false, Message: "Lỗi xử lý"}, err
	}

	// Dùng Set để lưu, key là ID, value là data
	err = s.RedisClient.SAdd(ctx, mempoolKey, txData.Bytes()).Err()
	if err != nil {
		log.Printf("Lỗi lưu tx vào mempool: %v", err)
		return &proto.Ack{Success: false, Message: "Lỗi mempool"}, err
	}

	// TODO: Lan truyền (broadcast) giao dịch này cho các node khác

	return &proto.Ack{Success: true, Message: "Đã nhận TX"}, nil
}

// AnnounceBlock (Khi 1 node khác thông báo đào được block)
func (s *Server) AnnounceBlock(ctx context.Context, req *proto.Block) (*proto.Ack, error) {
	log.Printf("Nhận được thông báo block mới: %x", req.Hash)

	// block := MapProtoBlockToDomain(req)

	// TODO:
	// 1. Xác thực block (PoW, chữ ký...)
	// 2. Kiểm tra xem có phải "longest chain" không
	// 3. Thêm block vào DB (s.Blockchain.AddBlock(...))
	// 4. Xóa các transaction của block này khỏi Mempool

	return &proto.Ack{Success: true, Message: "Đã nhận Block"}, nil
}

// GetBlocks (Node khác xin block)
func (s *Server) GetBlocks(req *proto.GetBlocksRequest, stream proto.NodeService_GetBlocksServer) error {
	log.Println("Nhận được yêu cầu đồng bộ (GetBlocks)")

	it := s.Blockchain.Iterator()

	for {
		block := it.Next()

		// Map và gửi block qua stream
		protoBlock := MapDomainBlockToProto(block)
		if err := stream.Send(protoBlock); err != nil {
			return err
		}

		// Dừng lại khi đến block Genesis
		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return nil
}

// GetKnownNodes (Node khác hỏi danh sách peer)
func (s *Server) GetKnownNodes(ctx context.Context, req *proto.EmptyRequest) (*proto.KnownNodesResponse, error) {
	// TODO: Triển khai logic quản lý danh sách node (Peer list)
	// Tạm thời trả về danh sách rỗng
	return &proto.KnownNodesResponse{Addresses: []string{}}, nil
}

// GetBalance (Triển khai API mới)
func (s *Server) GetBalance(ctx context.Context, req *proto.GetBalanceRequest) (*proto.GetBalanceResponse, error) {
	address := req.Address
	if !domain.ValidateAddress(address) {
		return nil, fmt.Errorf("địa chỉ ví không hợp lệ")
	}

	pubKeyHash := domain.DecodeAddress(address)

	// (Server có thể truy cập DB, nên nó sẽ chạy logic này)
	utxoSet := domain.UTXOSet{Blockchain: s.Blockchain}
	utxos := utxoSet.FindUTXO(pubKeyHash)

	var balance int64 = 0
	for _, out := range utxos {
		balance += out.Value
	}

	log.Printf("Đã truy vấn số dư cho %s: %d", address, balance)

	return &proto.GetBalanceResponse{Balance: balance}, nil
}

// (Dán hàm này vào network/server.go)

// FindSpendableUTXOs (Triển khai API mới)
func (s *Server) FindSpendableUTXOs(ctx context.Context, req *proto.FindSpendableUTXOsRequest) (*proto.FindSpendableUTXOsResponse, error) {
	if !domain.ValidateAddress(req.Address) {
		return nil, fmt.Errorf("địa chỉ không hợp lệ")
	}
	pubKeyHash := domain.DecodeAddress(req.Address)

	// Gọi hàm logic domain mới
	utxoSet := domain.UTXOSet{Blockchain: s.Blockchain}
	acc, spendableData := utxoSet.FindSpendableUTXOData(pubKeyHash, req.Amount)

	if acc < req.Amount {
		return nil, fmt.Errorf("không đủ tiền")
	}

	// Map kết quả sang kiểu proto
	var protoUTXOs []*proto.SpendableUTXO
	for _, utxo := range spendableData {
		protoUTXOs = append(protoUTXOs, &proto.SpendableUTXO{
			TxId:       utxo.TxID,
			VoutIndex:  int32(utxo.VoutIndex),
			Amount:     utxo.Amount,
			PubKeyHash: utxo.PubKeyHash,
		})
	}

	return &proto.FindSpendableUTXOsResponse{
		AccumulatedAmount: acc,
		Utxos:             protoUTXOs,
	}, nil
}

// (Dán vào cuối file network/server.go)

// GetContractState (Triển khai API mới)
func (s *Server) GetContractState(ctx context.Context, req *proto.GetContractStateRequest) (*proto.GetContractStateResponse, error) {
	log.Printf("Nhận được yêu cầu GetState cho contract: %s, key: %s", req.ContractAddress, req.Key)

	contractAddressBytes, err := hex.DecodeString(req.ContractAddress)
	if err != nil {
		return nil, fmt.Errorf("địa chỉ contract không hợp lệ")
	}

	// Gọi hàm domain
	value, err := s.Blockchain.GetContractState(contractAddressBytes, []byte(req.Key))
	if err != nil {
		return nil, fmt.Errorf("lỗi đọc CSDL: %v", err)
	}

	if value == nil {
		return &proto.GetContractStateResponse{Value: ""}, nil // Trả về chuỗi rỗng nếu không tìm thấy
	}

	return &proto.GetContractStateResponse{Value: string(value)}, nil
}
