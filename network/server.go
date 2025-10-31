package network

import (
	"bytes"
	"context"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"log"
	"net"

	"github.com/go-redis/redis/v8"
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/proto"
	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedNodeServiceServer
	Blockchain  *domain.Blockchain
	RedisClient *redis.Client
}

const (
	mempoolKey = "gochain:mempool"
)

func StartServer(port string, bc *domain.Blockchain, rdb *redis.Client) {

	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("Không thể lắng nghe trên cổng %s: %v", port, err)
	}

	s := grpc.NewServer()

	proto.RegisterNodeServiceServer(s, &Server{
		Blockchain:  bc,
		RedisClient: rdb,
	})

	log.Printf("gRPC Server đang lắng nghe tại %v", lis.Addr())
	if err := s.Serve(lis); err != nil {
		log.Fatalf("gRPC Server thất bại: %v", err)
	}
}

func (s *Server) SendTransaction(ctx context.Context, req *proto.Transaction) (*proto.Ack, error) {
	log.Printf("Nhận được giao dịch mới: %x", req.Id)

	tx := MapProtoTransactionToDomain(req)

	var txData bytes.Buffer
	enc := gob.NewEncoder(&txData)
	err := enc.Encode(tx)
	if err != nil {
		log.Printf("Lỗi serialize tx: %v", err)
		return &proto.Ack{Success: false, Message: "Lỗi xử lý"}, err
	}

	err = s.RedisClient.SAdd(ctx, mempoolKey, txData.Bytes()).Err()
	if err != nil {
		log.Printf("Lỗi lưu tx vào mempool: %v", err)
		return &proto.Ack{Success: false, Message: "Lỗi mempool"}, err
	}

	return &proto.Ack{Success: true, Message: "Đã nhận TX"}, nil
}

func (s *Server) AnnounceBlock(ctx context.Context, req *proto.Block) (*proto.Ack, error) {
	log.Printf("Nhận được thông báo block mới: %x", req.Hash)

	return &proto.Ack{Success: true, Message: "Đã nhận Block"}, nil
}

func (s *Server) GetBlocks(req *proto.GetBlocksRequest, stream proto.NodeService_GetBlocksServer) error {
	log.Println("Nhận được yêu cầu đồng bộ (GetBlocks)")

	it := s.Blockchain.Iterator()

	for {
		block := it.Next()

		protoBlock := MapDomainBlockToProto(block)
		if err := stream.Send(protoBlock); err != nil {
			return err
		}

		if len(block.PrevBlockHash) == 0 {
			break
		}
	}
	return nil
}

func (s *Server) GetKnownNodes(ctx context.Context, req *proto.EmptyRequest) (*proto.KnownNodesResponse, error) {

	return &proto.KnownNodesResponse{Addresses: []string{}}, nil
}

func (s *Server) GetBalance(ctx context.Context, req *proto.GetBalanceRequest) (*proto.GetBalanceResponse, error) {
	address := req.Address
	if !domain.ValidateAddress(address) {
		return nil, fmt.Errorf("địa chỉ ví không hợp lệ")
	}

	pubKeyHash := domain.DecodeAddress(address)

	utxoSet := domain.UTXOSet{Blockchain: s.Blockchain}
	utxos := utxoSet.FindUTXO(pubKeyHash)

	var balance int64 = 0
	for _, out := range utxos {
		balance += out.Value
	}

	log.Printf("Đã truy vấn số dư cho %s: %d", address, balance)

	return &proto.GetBalanceResponse{Balance: balance}, nil
}

func (s *Server) FindSpendableUTXOs(ctx context.Context, req *proto.FindSpendableUTXOsRequest) (*proto.FindSpendableUTXOsResponse, error) {
	if !domain.ValidateAddress(req.Address) {
		return nil, fmt.Errorf("địa chỉ không hợp lệ")
	}
	pubKeyHash := domain.DecodeAddress(req.Address)

	utxoSet := domain.UTXOSet{Blockchain: s.Blockchain}
	acc, spendableData := utxoSet.FindSpendableUTXOData(pubKeyHash, req.Amount)

	if acc < req.Amount {
		return nil, fmt.Errorf("không đủ tiền")
	}

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

func (s *Server) GetContractState(ctx context.Context, req *proto.GetContractStateRequest) (*proto.GetContractStateResponse, error) {
	log.Printf("Nhận được yêu cầu GetState cho contract: %s, key: %s", req.ContractAddress, req.Key)

	contractAddressBytes, err := hex.DecodeString(req.ContractAddress)
	if err != nil {
		return nil, fmt.Errorf("địa chỉ contract không hợp lệ")
	}

	value, err := s.Blockchain.GetContractState(contractAddressBytes, []byte(req.Key))
	if err != nil {
		return nil, fmt.Errorf("lỗi đọc CSDL: %v", err)
	}

	if value == nil {
		return &proto.GetContractStateResponse{Value: ""}, nil
	}

	return &proto.GetContractStateResponse{Value: string(value)}, nil
}
