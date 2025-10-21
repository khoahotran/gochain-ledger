package network

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/proto"
)

// PublicServer triển khai logic cho DApp
type PublicServer struct {
	proto.UnimplementedPublicServiceServer // Bắt buộc
	Blockchain                             *domain.Blockchain
	RedisClient                            *redis.Client
}

// GetBalance (Copy từ server.go)
func (s *PublicServer) GetBalance(ctx context.Context, req *proto.GetBalanceRequest) (*proto.GetBalanceResponse, error) {
	// (Logic này giống hệt server.go, chỉ đổi tên struct)
	gs := &Server{Blockchain: s.Blockchain}
	return gs.GetBalance(ctx, req)
}

// GetContractState (Copy từ server.go)
func (s *PublicServer) GetContractState(ctx context.Context, req *proto.GetContractStateRequest) (*proto.GetContractStateResponse, error) {
	// (Logic này giống hệt server.go)
	gs := &Server{Blockchain: s.Blockchain}
	return gs.GetContractState(ctx, req)
}

// SubmitTransaction (Gửi TX từ DApp)
func (s *PublicServer) SubmitTransaction(ctx context.Context, req *proto.Transaction) (*proto.Ack, error) {
	// DApp đã ký TX ở client-side, nên chúng ta chỉ cần
	// chuyển nó cho logic 'SendTransaction' P2P (để verify và đưa vào mempool)
	gs := &Server{Blockchain: s.Blockchain, RedisClient: s.RedisClient}
	return gs.SendTransaction(ctx, req)
}

// FindSpendableUTXOs (Triển khai API mới cho Public)
func (s *PublicServer) FindSpendableUTXOs(ctx context.Context, req *proto.FindSpendableUTXOsRequest) (*proto.FindSpendableUTXOsResponse, error) {
	// Chúng ta có thể gọi thẳng logic tương tự trong Server P2P
	gs := &Server{Blockchain: s.Blockchain} // Chỉ cần blockchain để đọc
	return gs.FindSpendableUTXOs(ctx, req)
}
