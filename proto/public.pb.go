package proto

import (
	reflect "reflect"
	unsafe "unsafe"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)

	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

var File_proto_public_proto protoreflect.FileDescriptor

const file_proto_public_proto_rawDesc = "" +
	"\n" +
	"\x12proto/public.proto\x12\x05proto\x1a\x16proto/blockchain.proto2\xb7\x02\n" +
	"\rPublicService\x12A\n" +
	"\n" +
	"GetBalance\x12\x18.proto.GetBalanceRequest\x1a\x19.proto.GetBalanceResponse\x12S\n" +
	"\x10GetContractState\x12\x1e.proto.GetContractStateRequest\x1a\x1f.proto.GetContractStateResponse\x123\n" +
	"\x11SubmitTransaction\x12\x12.proto.Transaction\x1a\n" +
	".proto.Ack\x12Y\n" +
	"\x12FindSpendableUTXOs\x12 .proto.FindSpendableUTXOsRequest\x1a!.proto.FindSpendableUTXOsResponseB\tZ\a./protob\x06proto3"

var file_proto_public_proto_goTypes = []any{
	(*GetBalanceRequest)(nil),
	(*GetContractStateRequest)(nil),
	(*Transaction)(nil),
	(*FindSpendableUTXOsRequest)(nil),
	(*GetBalanceResponse)(nil),
	(*GetContractStateResponse)(nil),
	(*Ack)(nil),
	(*FindSpendableUTXOsResponse)(nil),
}
var file_proto_public_proto_depIdxs = []int32{
	0,
	1,
	2,
	3,
	4,
	5,
	6,
	7,
	4,
	0,
	0,
	0,
	0,
}

func init() { file_proto_public_proto_init() }
func file_proto_public_proto_init() {
	if File_proto_public_proto != nil {
		return
	}
	file_proto_blockchain_proto_init()
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: unsafe.Slice(unsafe.StringData(file_proto_public_proto_rawDesc), len(file_proto_public_proto_rawDesc)),
			NumEnums:      0,
			NumMessages:   0,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_proto_public_proto_goTypes,
		DependencyIndexes: file_proto_public_proto_depIdxs,
	}.Build()
	File_proto_public_proto = out.File
	file_proto_public_proto_goTypes = nil
	file_proto_public_proto_depIdxs = nil
}
