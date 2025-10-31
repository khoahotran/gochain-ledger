package proto

import (
	context "context"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

const _ = grpc.SupportPackageIsVersion9

const (
	PublicService_GetBalance_FullMethodName         = "/proto.PublicService/GetBalance"
	PublicService_GetContractState_FullMethodName   = "/proto.PublicService/GetContractState"
	PublicService_SubmitTransaction_FullMethodName  = "/proto.PublicService/SubmitTransaction"
	PublicService_FindSpendableUTXOs_FullMethodName = "/proto.PublicService/FindSpendableUTXOs"
)
type PublicServiceClient interface {
	GetBalance(ctx context.Context, in *GetBalanceRequest, opts ...grpc.CallOption) (*GetBalanceResponse, error)

	GetContractState(ctx context.Context, in *GetContractStateRequest, opts ...grpc.CallOption) (*GetContractStateResponse, error)

	SubmitTransaction(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*Ack, error)

	FindSpendableUTXOs(ctx context.Context, in *FindSpendableUTXOsRequest, opts ...grpc.CallOption) (*FindSpendableUTXOsResponse, error)
}

type publicServiceClient struct {
	cc grpc.ClientConnInterface
}

func NewPublicServiceClient(cc grpc.ClientConnInterface) PublicServiceClient {
	return &publicServiceClient{cc}
}

func (c *publicServiceClient) GetBalance(ctx context.Context, in *GetBalanceRequest, opts ...grpc.CallOption) (*GetBalanceResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetBalanceResponse)
	err := c.cc.Invoke(ctx, PublicService_GetBalance_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *publicServiceClient) GetContractState(ctx context.Context, in *GetContractStateRequest, opts ...grpc.CallOption) (*GetContractStateResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(GetContractStateResponse)
	err := c.cc.Invoke(ctx, PublicService_GetContractState_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *publicServiceClient) SubmitTransaction(ctx context.Context, in *Transaction, opts ...grpc.CallOption) (*Ack, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(Ack)
	err := c.cc.Invoke(ctx, PublicService_SubmitTransaction_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *publicServiceClient) FindSpendableUTXOs(ctx context.Context, in *FindSpendableUTXOsRequest, opts ...grpc.CallOption) (*FindSpendableUTXOsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(FindSpendableUTXOsResponse)
	err := c.cc.Invoke(ctx, PublicService_FindSpendableUTXOs_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

type PublicServiceServer interface {
	GetBalance(context.Context, *GetBalanceRequest) (*GetBalanceResponse, error)

	GetContractState(context.Context, *GetContractStateRequest) (*GetContractStateResponse, error)

	SubmitTransaction(context.Context, *Transaction) (*Ack, error)

	FindSpendableUTXOs(context.Context, *FindSpendableUTXOsRequest) (*FindSpendableUTXOsResponse, error)
	mustEmbedUnimplementedPublicServiceServer()
}

type UnimplementedPublicServiceServer struct{}

func (UnimplementedPublicServiceServer) GetBalance(context.Context, *GetBalanceRequest) (*GetBalanceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetBalance not implemented")
}
func (UnimplementedPublicServiceServer) GetContractState(context.Context, *GetContractStateRequest) (*GetContractStateResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method GetContractState not implemented")
}
func (UnimplementedPublicServiceServer) SubmitTransaction(context.Context, *Transaction) (*Ack, error) {
	return nil, status.Errorf(codes.Unimplemented, "method SubmitTransaction not implemented")
}
func (UnimplementedPublicServiceServer) FindSpendableUTXOs(context.Context, *FindSpendableUTXOsRequest) (*FindSpendableUTXOsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method FindSpendableUTXOs not implemented")
}
func (UnimplementedPublicServiceServer) mustEmbedUnimplementedPublicServiceServer() {}
func (UnimplementedPublicServiceServer) testEmbeddedByValue()                       {}

type UnsafePublicServiceServer interface {
	mustEmbedUnimplementedPublicServiceServer()
}

func RegisterPublicServiceServer(s grpc.ServiceRegistrar, srv PublicServiceServer) {

	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&PublicService_ServiceDesc, srv)
}

func _PublicService_GetBalance_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBalanceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PublicServiceServer).GetBalance(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PublicService_GetBalance_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PublicServiceServer).GetBalance(ctx, req.(*GetBalanceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PublicService_GetContractState_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetContractStateRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PublicServiceServer).GetContractState(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PublicService_GetContractState_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PublicServiceServer).GetContractState(ctx, req.(*GetContractStateRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _PublicService_SubmitTransaction_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Transaction)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PublicServiceServer).SubmitTransaction(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PublicService_SubmitTransaction_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PublicServiceServer).SubmitTransaction(ctx, req.(*Transaction))
	}
	return interceptor(ctx, in, info, handler)
}

func _PublicService_FindSpendableUTXOs_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FindSpendableUTXOsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(PublicServiceServer).FindSpendableUTXOs(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: PublicService_FindSpendableUTXOs_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(PublicServiceServer).FindSpendableUTXOs(ctx, req.(*FindSpendableUTXOsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

var PublicService_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "proto.PublicService",
	HandlerType: (*PublicServiceServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "GetBalance",
			Handler:    _PublicService_GetBalance_Handler,
		},
		{
			MethodName: "GetContractState",
			Handler:    _PublicService_GetContractState_Handler,
		},
		{
			MethodName: "SubmitTransaction",
			Handler:    _PublicService_SubmitTransaction_Handler,
		},
		{
			MethodName: "FindSpendableUTXOs",
			Handler:    _PublicService_FindSpendableUTXOs_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "proto/public.proto",
}
