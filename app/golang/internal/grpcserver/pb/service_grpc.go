package pb

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// This file hand-mirrors what `protoc-gen-go-grpc` would emit for the
// `NodeGateway` service in sms.proto: a client interface + implementation,
// a server interface, and a grpc.ServiceDesc wiring method names to
// handlers. See the note at the top of messages.go for why it's
// hand-written instead of generated.

const (
	serviceName = "openchat.v1.NodeGateway"

	methodGetAddress        = "/" + serviceName + "/GetAddress"
	methodSendSMS           = "/" + serviceName + "/SendSMS"
	methodStreamIncomingSMS = "/" + serviceName + "/StreamIncomingSMS"
	methodGetNodesDiscovery = "/" + serviceName + "/GetNodesDiscovery"
	methodRegisterRelay     = "/" + serviceName + "/RegisterRelay"
	methodGetBlocks         = "/" + serviceName + "/GetBlocks"
)

// ---------------------------------------------------------------------
// Server side
// ---------------------------------------------------------------------

// NodeGatewayServer is the interface node-side implementations satisfy.
type NodeGatewayServer interface {
	GetAddress(context.Context, *Empty) (*AddressResponse, error)
	SendSMS(context.Context, *SMSRequest) (*SMSResponse, error)
	StreamIncomingSMS(*StreamRequest, NodeGateway_StreamIncomingSMSServer) error
	GetNodesDiscovery(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error)
	RegisterRelay(context.Context, *RegisterRelayRequest) (*RegisterRelayResponse, error)
	GetBlocks(context.Context, *GetBlocksRequest) (*GetBlocksResponse, error)
}

// UnimplementedNodeGatewayServer can be embedded for forward compatibility.
type UnimplementedNodeGatewayServer struct{}

func (UnimplementedNodeGatewayServer) GetAddress(context.Context, *Empty) (*AddressResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetAddress not implemented")
}
func (UnimplementedNodeGatewayServer) SendSMS(context.Context, *SMSRequest) (*SMSResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method SendSMS not implemented")
}
func (UnimplementedNodeGatewayServer) StreamIncomingSMS(*StreamRequest, NodeGateway_StreamIncomingSMSServer) error {
	return status.Error(codes.Unimplemented, "method StreamIncomingSMS not implemented")
}
func (UnimplementedNodeGatewayServer) GetNodesDiscovery(context.Context, *DiscoveryRequest) (*DiscoveryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetNodesDiscovery not implemented")
}
func (UnimplementedNodeGatewayServer) RegisterRelay(context.Context, *RegisterRelayRequest) (*RegisterRelayResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method RegisterRelay not implemented")
}
func (UnimplementedNodeGatewayServer) GetBlocks(context.Context, *GetBlocksRequest) (*GetBlocksResponse, error) {
	return nil, status.Error(codes.Unimplemented, "method GetBlocks not implemented")
}

// NodeGateway_StreamIncomingSMSServer is the server-side stream handle
// passed into StreamIncomingSMS implementations.
type NodeGateway_StreamIncomingSMSServer interface {
	Send(*SMSResponse) error
	grpc.ServerStream
}

type nodeGatewayStreamIncomingSMSServer struct {
	grpc.ServerStream
}

func (s *nodeGatewayStreamIncomingSMSServer) Send(m *SMSResponse) error {
	return s.ServerStream.SendMsg(m)
}

func RegisterNodeGatewayServer(s grpc.ServiceRegistrar, srv NodeGatewayServer) {
	s.RegisterService(&NodeGateway_ServiceDesc, srv)
}

func _NodeGateway_GetAddress_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeGatewayServer).GetAddress(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: methodGetAddress}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeGatewayServer).GetAddress(ctx, req.(*Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeGateway_SendSMS_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SMSRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeGatewayServer).SendSMS(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: methodSendSMS}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeGatewayServer).SendSMS(ctx, req.(*SMSRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeGateway_StreamIncomingSMS_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(NodeGatewayServer).StreamIncomingSMS(m, &nodeGatewayStreamIncomingSMSServer{stream})
}

func _NodeGateway_GetNodesDiscovery_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DiscoveryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeGatewayServer).GetNodesDiscovery(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: methodGetNodesDiscovery}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeGatewayServer).GetNodesDiscovery(ctx, req.(*DiscoveryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeGateway_RegisterRelay_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(RegisterRelayRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeGatewayServer).RegisterRelay(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: methodRegisterRelay}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeGatewayServer).RegisterRelay(ctx, req.(*RegisterRelayRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _NodeGateway_GetBlocks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(GetBlocksRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeGatewayServer).GetBlocks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{Server: srv, FullMethod: methodGetBlocks}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeGatewayServer).GetBlocks(ctx, req.(*GetBlocksRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// NodeGateway_ServiceDesc is the grpc.ServiceDesc for NodeGateway.
var NodeGateway_ServiceDesc = grpc.ServiceDesc{
	ServiceName: serviceName,
	HandlerType: (*NodeGatewayServer)(nil),
	Methods: []grpc.MethodDesc{
		{MethodName: "GetAddress", Handler: _NodeGateway_GetAddress_Handler},
		{MethodName: "SendSMS", Handler: _NodeGateway_SendSMS_Handler},
		{MethodName: "GetNodesDiscovery", Handler: _NodeGateway_GetNodesDiscovery_Handler},
		{MethodName: "RegisterRelay", Handler: _NodeGateway_RegisterRelay_Handler},
		{MethodName: "GetBlocks", Handler: _NodeGateway_GetBlocks_Handler},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamIncomingSMS",
			Handler:       _NodeGateway_StreamIncomingSMS_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "api/protobuf/sms.proto",
}

// ---------------------------------------------------------------------
// Client side
// ---------------------------------------------------------------------

// NodeGatewayClient is the interface client-side code (pkg/client) uses.
type NodeGatewayClient interface {
	GetAddress(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*AddressResponse, error)
	SendSMS(ctx context.Context, in *SMSRequest, opts ...grpc.CallOption) (*SMSResponse, error)
	StreamIncomingSMS(ctx context.Context, in *StreamRequest, opts ...grpc.CallOption) (NodeGateway_StreamIncomingSMSClient, error)
	GetNodesDiscovery(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error)
	RegisterRelay(ctx context.Context, in *RegisterRelayRequest, opts ...grpc.CallOption) (*RegisterRelayResponse, error)
	GetBlocks(ctx context.Context, in *GetBlocksRequest, opts ...grpc.CallOption) (*GetBlocksResponse, error)
}

type nodeGatewayClient struct {
	cc grpc.ClientConnInterface
}

// NewNodeGatewayClient wraps an established *grpc.ClientConn.
func NewNodeGatewayClient(cc grpc.ClientConnInterface) NodeGatewayClient {
	return &nodeGatewayClient{cc}
}

func (c *nodeGatewayClient) GetAddress(ctx context.Context, in *Empty, opts ...grpc.CallOption) (*AddressResponse, error) {
	out := new(AddressResponse)
	if err := c.cc.Invoke(ctx, methodGetAddress, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeGatewayClient) SendSMS(ctx context.Context, in *SMSRequest, opts ...grpc.CallOption) (*SMSResponse, error) {
	out := new(SMSResponse)
	if err := c.cc.Invoke(ctx, methodSendSMS, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeGatewayClient) GetNodesDiscovery(ctx context.Context, in *DiscoveryRequest, opts ...grpc.CallOption) (*DiscoveryResponse, error) {
	out := new(DiscoveryResponse)
	if err := c.cc.Invoke(ctx, methodGetNodesDiscovery, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeGatewayClient) RegisterRelay(ctx context.Context, in *RegisterRelayRequest, opts ...grpc.CallOption) (*RegisterRelayResponse, error) {
	out := new(RegisterRelayResponse)
	if err := c.cc.Invoke(ctx, methodRegisterRelay, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeGatewayClient) GetBlocks(ctx context.Context, in *GetBlocksRequest, opts ...grpc.CallOption) (*GetBlocksResponse, error) {
	out := new(GetBlocksResponse)
	if err := c.cc.Invoke(ctx, methodGetBlocks, in, out, opts...); err != nil {
		return nil, err
	}
	return out, nil
}

// NodeGateway_StreamIncomingSMSClient is the client-side stream handle
// returned by StreamIncomingSMS.
type NodeGateway_StreamIncomingSMSClient interface {
	Recv() (*SMSResponse, error)
	grpc.ClientStream
}

type nodeGatewayStreamIncomingSMSClient struct {
	grpc.ClientStream
}

func (x *nodeGatewayStreamIncomingSMSClient) Recv() (*SMSResponse, error) {
	m := new(SMSResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *nodeGatewayClient) StreamIncomingSMS(ctx context.Context, in *StreamRequest, opts ...grpc.CallOption) (NodeGateway_StreamIncomingSMSClient, error) {
	stream, err := c.cc.NewStream(ctx, &NodeGateway_ServiceDesc.Streams[0], methodStreamIncomingSMS, opts...)
	if err != nil {
		return nil, err
	}
	x := &nodeGatewayStreamIncomingSMSClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}
