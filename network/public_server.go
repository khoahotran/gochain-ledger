package network

import (
	"context"

	"github.com/go-redis/redis/v8"
	"github.com/khoahotran/gochain-ledger/domain"
	"github.com/khoahotran/gochain-ledger/proto"
)

type PublicServer struct {
	proto.UnimplementedPublicServiceServer
	Blockchain  *domain.Blockchain
	RedisClient *redis.Client
}

func (s *PublicServer) GetBalance(ctx context.Context, req *proto.GetBalanceRequest) (*proto.GetBalanceResponse, error) {

	gs := &Server{Blockchain: s.Blockchain}
	return gs.GetBalance(ctx, req)
}

func (s *PublicServer) GetContractState(ctx context.Context, req *proto.GetContractStateRequest) (*proto.GetContractStateResponse, error) {

	gs := &Server{Blockchain: s.Blockchain}
	return gs.GetContractState(ctx, req)
}

func (s *PublicServer) SubmitTransaction(ctx context.Context, req *proto.Transaction) (*proto.Ack, error) {

	gs := &Server{Blockchain: s.Blockchain, RedisClient: s.RedisClient}
	return gs.SendTransaction(ctx, req)
}

func (s *PublicServer) FindSpendableUTXOs(ctx context.Context, req *proto.FindSpendableUTXOsRequest) (*proto.FindSpendableUTXOsResponse, error) {

	gs := &Server{Blockchain: s.Blockchain}
	return gs.FindSpendableUTXOs(ctx, req)
}
