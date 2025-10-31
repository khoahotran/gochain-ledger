package proto

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const _ = grpc.SupportPackageIsVersion9

const (
	NodeService_SendTransaction_FullMethodName    = "/proto.NodeService/SendTransaction"
	NodeService_AnnounceBlock_FullMethodName      = "/proto.NodeService/AnnounceBlock"
	NodeService_GetBlocks_FullMethodName          = "/proto.NodeService/GetBlocks"
	NodeService_GetKnownNodes_FullMethodName      = "/proto.NodeService/GetKnownNodes"
	NodeService_GetBalance_FullMethodName         = "/proto.NodeService/GetBalance"
	NodeService_FindSpendableUTXOs_FullMethodName = "/proto.NodeService/FindSpendableUTXOs"
	NodeService_GetContractState_FullMethodName   = "/proto.NodeService/GetContractState"
)

type NodeServiceClient interface {
	SendTransaction(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*Ack, error)

	AnnounceBlock(ctx context.Context, in *Block, opts ...grpc.CallOption) (*Ack, error)

	GetBlocks(ctx context.Context, in *GetBlocksRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Block], error)

	GetKnownNodes(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*KnownNodesResponse, error)

	GetBalance(ctx context.Context, in *GetBalanceRequest, opts ...grpc.CallOption) (*GetBalanceResponse, error)

	FindSpendableUTXOs(ctx context.Context, in *FindSpendableUTXOsRequest, opts ...grpc.CallOption) (*FindSpendableUTXOsResponse, error)

	GetContractState(ctx context.Context, in *GetContractStateRequest, opts ...grpc.CallOption) (*GetContractStateResponse, error)
}

type nodeServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeServiceClient(cc grpc.ClientConnInterface) NodeServiceClient {
	return &nodeServiceClient{cc}
}

func (c *nodeServiceClient) SendTransaction(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*Ack, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Ack)
	err := c.cc.Invoke(ctx, NodeService_SendTransaction_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) AnnounceBlock(ctx context.Context, in *Block, opts ...grpc.CallOption) (*Ack, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Ack)
	err := c.cc.Invoke(ctx, NodeService_AnnounceBlock_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) GetBlocks(ctx context.Context, in *GetBlocksRequest, opts ...grpc.CallOption) (grpc.ServerStreamingClient[Block], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &NodeService_ServiceDesc.Streams[0], NodeService_GetBlocks_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[GetBlocksRequest, Block]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type NodeService_GetBlocksClient = grpc.ServerStreamingClient[Block]

func (c *nodeServiceClient) GetKnownNodes(ctx context.Context, in *EmptyRequest, opts ...grpc.CallOption) (*KnownNodesResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(KnownNodesResponse)
	err := c.cc.Invoke(ctx, NodeService_GetKnownNodes_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) GetBalance(ctx context.Context, in *GetBalanceRequest, opts ...grpc.CallOption) (*GetBalanceResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetBalanceResponse)
	err := c.cc.Invoke(ctx, NodeService_GetBalance_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) FindSpendableUTXOs(ctx context.Context, in *FindSpendableUTXOsRequest, opts ...grpc.CallOption) (*FindSpendableUTXOsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(FindSpendableUTXOsResponse)
	err := c.cc.Invoke(ctx, NodeService_FindSpendableUTXOs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeServiceClient) GetContractState(ctx context.Context, in *GetContractStateRequest, opts ...grpc.CallOption) (*GetContractStateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetContractStateResponse)
	err := c.cc.Invoke(ctx, NodeService_GetContractState_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type NodeServiceServer interface {
	SendTransaction(context.Context, *Transaction) (*Ack, error)

	AnnounceBlock(context.Context, *Block) (*Ack, error)

	GetBlocks(*GetBlocksRequest, grpc.ServerStreamingServer[Block]) error

	GetKnownNodes(context.Context, *EmptyRequest) (*KnownNodesResponse, error)

	GetBalance(context.Context, *GetBalanceRequest) (*GetBalanceResponse, error)

	FindSpendableUTXOs(context.Context, *FindSpendableUTXOsRequest) (*FindSpendableUTXOsResponse, error)

	GetContractState(context.Context, *GetContractStateRequest) (*GetContractStateResponse, error)
	mustEmbedUnimplementedNodeServiceServer()
}

type UnimplementedNodeServiceServer struct{}

func (UnimplementedNodeServiceServer) SendTransaction(context.Context, *Transaction) (*Ack, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SendTransaction not implemented")
}
func (UnimplementedNodeServiceServer) AnnounceBlock(context.Context, *Block) (*Ack, error) {
	return nil, status.Errorf(codes.Unimplemented, "method AnnounceBlock not implemented")
}
func (UnimplementedNodeServiceServer) GetBlocks(*GetBlocksRequest, grpc.ServerStreamingServer[Block]) error {
	return status.Errorf(codes.Unimplemented, "method GetBlocks not implemented")
}
func (UnimplementedNodeServiceServer) GetKnownNodes(context.Context, *EmptyRequest) (*KnownNodesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetKnownNodes not implemented")
}
func (UnimplementedNodeServiceServer) GetBalance(context.Context, *GetBalanceRequest) (*GetBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBalance not implemented")
}
func (UnimplementedNodeServiceServer) FindSpendableUTXOs(context.Context, *FindSpendableUTXOsRequest) (*FindSpendableUTXOsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FindSpendableUTXOs not implemented")
}
func (UnimplementedNodeServiceServer) GetContractState(context.Context, *GetContractStateRequest) (*GetContractStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContractState not implemented")
}
func (UnimplementedNodeServiceServer) mustEmbedUnimplementedNodeServiceServer() {}
func (UnimplementedNodeServiceServer) testEmbeddedByValue()                     {}

type UnsafeNodeServiceServer interface {
	mustEmbedUnimplementedNodeServiceServer()
}

func RegisterNodeServiceServer(s grpc.ServiceRegistrar, srv NodeServiceServer) {

	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&NodeService_ServiceDesc, srv)
}

func _NodeService_SendTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Transaction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).SendTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NodeService_SendTransaction_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).SendTransaction(ctx, req.(*Transaction))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_AnnounceBlock_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Block)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).AnnounceBlock(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NodeService_AnnounceBlock_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).AnnounceBlock(ctx, req.(*Block))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_GetBlocks_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(GetBlocksRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(NodeServiceServer).GetBlocks(m, &grpc.GenericServerStream[GetBlocksRequest, Block]{ServerStream: stream})
}

type NodeService_GetBlocksServer = grpc.ServerStreamingServer[Block]

func _NodeService_GetKnownNodes_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(EmptyRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).GetKnownNodes(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NodeService_GetKnownNodes_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).GetKnownNodes(ctx, req.(*EmptyRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_GetBalance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).GetBalance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NodeService_GetBalance_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).GetBalance(ctx, req.(*GetBalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_FindSpendableUTXOs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FindSpendableUTXOsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).FindSpendableUTXOs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NodeService_FindSpendableUTXOs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).FindSpendableUTXOs(ctx, req.(*FindSpendableUTXOsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeService_GetContractState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContractStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServiceServer).GetContractState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: NodeService_GetContractState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServiceServer).GetContractState(ctx, req.(*GetContractStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var NodeService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.NodeService",
	HandlerType: (*NodeServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "SendTransaction",
			Handler:    _NodeService_SendTransaction_Handler,
		},
		{
			MethodName: "AnnounceBlock",
			Handler:    _NodeService_AnnounceBlock_Handler,
		},
		{
			MethodName: "GetKnownNodes",
			Handler:    _NodeService_GetKnownNodes_Handler,
		},
		{
			MethodName: "GetBalance",
			Handler:    _NodeService_GetBalance_Handler,
		},
		{
			MethodName: "FindSpendableUTXOs",
			Handler:    _NodeService_FindSpendableUTXOs_Handler,
		},
		{
			MethodName: "GetContractState",
			Handler:    _NodeService_GetContractState_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "GetBlocks",
			Handler:       _NodeService_GetBlocks_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "proto/blockchain.proto",
}
