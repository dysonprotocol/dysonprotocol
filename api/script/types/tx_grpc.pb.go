// Since: cosmos-sdk 0.46

// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             (unknown)
// source: dysonprotocol/script/v1/tx.proto

package types

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Msg_UpdateScript_FullMethodName    = "/dysonprotocol.script.v1.Msg/UpdateScript"
	Msg_ExecScript_FullMethodName      = "/dysonprotocol.script.v1.Msg/ExecScript"
	Msg_CreateNewScript_FullMethodName = "/dysonprotocol.script.v1.Msg/CreateNewScript"
	Msg_UpdateParams_FullMethodName    = "/dysonprotocol.script.v1.Msg/UpdateParams"
)

// MsgClient is the client API for Msg service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// Msg is the dysonprotocol.script.v1 Msg service.
type MsgClient interface {
	// Update Script updates the script of the given address and increments the
	// the version.
	UpdateScript(ctx context.Context, in *MsgUpdateScript, opts ...grpc.CallOption) (*MsgUpdateScriptResponse, error)
	// ExecScript executes a script by calling a function with arguments.
	ExecScript(ctx context.Context, in *MsgExec, opts ...grpc.CallOption) (*MsgExecResponse, error)
	// Create a new script with new address = hash(creator + content), a new authz
	// updatescript permission will also be created for the creator
	CreateNewScript(ctx context.Context, in *MsgCreateNewScript, opts ...grpc.CallOption) (*MsgCreateNewScriptResponse, error)
	// UpdateParams defines a governance operation for updating the x/script
	// module parameters. The authority defaults to the x/gov module account.
	UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error)
}

type msgClient struct {
	cc grpc.ClientConnInterface
}

func NewMsgClient(cc grpc.ClientConnInterface) MsgClient {
	return &msgClient{cc}
}

func (c *msgClient) UpdateScript(ctx context.Context, in *MsgUpdateScript, opts ...grpc.CallOption) (*MsgUpdateScriptResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgUpdateScriptResponse)
	err := c.cc.Invoke(ctx, Msg_UpdateScript_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) ExecScript(ctx context.Context, in *MsgExec, opts ...grpc.CallOption) (*MsgExecResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgExecResponse)
	err := c.cc.Invoke(ctx, Msg_ExecScript_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) CreateNewScript(ctx context.Context, in *MsgCreateNewScript, opts ...grpc.CallOption) (*MsgCreateNewScriptResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgCreateNewScriptResponse)
	err := c.cc.Invoke(ctx, Msg_CreateNewScript_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *msgClient) UpdateParams(ctx context.Context, in *MsgUpdateParams, opts ...grpc.CallOption) (*MsgUpdateParamsResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(MsgUpdateParamsResponse)
	err := c.cc.Invoke(ctx, Msg_UpdateParams_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// MsgServer is the server API for Msg service.
// All implementations must embed UnimplementedMsgServer
// for forward compatibility.
//
// Msg is the dysonprotocol.script.v1 Msg service.
type MsgServer interface {
	// Update Script updates the script of the given address and increments the
	// the version.
	UpdateScript(context.Context, *MsgUpdateScript) (*MsgUpdateScriptResponse, error)
	// ExecScript executes a script by calling a function with arguments.
	ExecScript(context.Context, *MsgExec) (*MsgExecResponse, error)
	// Create a new script with new address = hash(creator + content), a new authz
	// updatescript permission will also be created for the creator
	CreateNewScript(context.Context, *MsgCreateNewScript) (*MsgCreateNewScriptResponse, error)
	// UpdateParams defines a governance operation for updating the x/script
	// module parameters. The authority defaults to the x/gov module account.
	UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error)
	mustEmbedUnimplementedMsgServer()
}

// UnimplementedMsgServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedMsgServer struct{}

func (UnimplementedMsgServer) UpdateScript(context.Context, *MsgUpdateScript) (*MsgUpdateScriptResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateScript not implemented")
}
func (UnimplementedMsgServer) ExecScript(context.Context, *MsgExec) (*MsgExecResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ExecScript not implemented")
}
func (UnimplementedMsgServer) CreateNewScript(context.Context, *MsgCreateNewScript) (*MsgCreateNewScriptResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method CreateNewScript not implemented")
}
func (UnimplementedMsgServer) UpdateParams(context.Context, *MsgUpdateParams) (*MsgUpdateParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateParams not implemented")
}
func (UnimplementedMsgServer) mustEmbedUnimplementedMsgServer() {}
func (UnimplementedMsgServer) testEmbeddedByValue()             {}

// UnsafeMsgServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to MsgServer will
// result in compilation errors.
type UnsafeMsgServer interface {
	mustEmbedUnimplementedMsgServer()
}

func RegisterMsgServer(s grpc.ServiceRegistrar, srv MsgServer) {
	// If the following call pancis, it indicates UnimplementedMsgServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Msg_ServiceDesc, srv)
}

func _Msg_UpdateScript_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateScript)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateScript(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_UpdateScript_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateScript(ctx, req.(*MsgUpdateScript))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_ExecScript_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgExec)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).ExecScript(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_ExecScript_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).ExecScript(ctx, req.(*MsgExec))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_CreateNewScript_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgCreateNewScript)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).CreateNewScript(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_CreateNewScript_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).CreateNewScript(ctx, req.(*MsgCreateNewScript))
	}
	return interceptor(ctx, in, info, handler)
}

func _Msg_UpdateParams_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(MsgUpdateParams)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(MsgServer).UpdateParams(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Msg_UpdateParams_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(MsgServer).UpdateParams(ctx, req.(*MsgUpdateParams))
	}
	return interceptor(ctx, in, info, handler)
}

// Msg_ServiceDesc is the grpc.ServiceDesc for Msg service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Msg_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "dysonprotocol.script.v1.Msg",
	HandlerType: (*MsgServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateScript",
			Handler:    _Msg_UpdateScript_Handler,
		},
		{
			MethodName: "ExecScript",
			Handler:    _Msg_ExecScript_Handler,
		},
		{
			MethodName: "CreateNewScript",
			Handler:    _Msg_CreateNewScript_Handler,
		},
		{
			MethodName: "UpdateParams",
			Handler:    _Msg_UpdateParams_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "dysonprotocol/script/v1/tx.proto",
}
