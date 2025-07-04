// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: dysonprotocol/script/v1/events.proto

package types

import (
	fmt "fmt"
	_ "github.com/cosmos/cosmos-proto"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	_ "github.com/cosmos/gogoproto/types/any"
	io "io"
	math "math"
	math_bits "math/bits"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// EventUpdateScript is an event emitted when a script is updated.
type EventUpdateScript struct {
	// The new version of the script.
	Version uint64 `protobuf:"varint,1,opt,name=version,proto3" json:"version,omitempty"`
}

func (m *EventUpdateScript) Reset()         { *m = EventUpdateScript{} }
func (m *EventUpdateScript) String() string { return proto.CompactTextString(m) }
func (*EventUpdateScript) ProtoMessage()    {}
func (*EventUpdateScript) Descriptor() ([]byte, []int) {
	return fileDescriptor_848ca6ed468fa3ce, []int{0}
}
func (m *EventUpdateScript) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventUpdateScript) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EventUpdateScript.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EventUpdateScript) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventUpdateScript.Merge(m, src)
}
func (m *EventUpdateScript) XXX_Size() int {
	return m.Size()
}
func (m *EventUpdateScript) XXX_DiscardUnknown() {
	xxx_messageInfo_EventUpdateScript.DiscardUnknown(m)
}

var xxx_messageInfo_EventUpdateScript proto.InternalMessageInfo

func (m *EventUpdateScript) GetVersion() uint64 {
	if m != nil {
		return m.Version
	}
	return 0
}

// Event ExecScript is an event emitted when a script is executed.
type EventExecScript struct {
	// The result of the script execution.
	Request  *MsgExec         `protobuf:"bytes,1,opt,name=request,proto3" json:"request,omitempty"`
	Response *MsgExecResponse `protobuf:"bytes,2,opt,name=response,proto3" json:"response,omitempty"`
}

func (m *EventExecScript) Reset()         { *m = EventExecScript{} }
func (m *EventExecScript) String() string { return proto.CompactTextString(m) }
func (*EventExecScript) ProtoMessage()    {}
func (*EventExecScript) Descriptor() ([]byte, []int) {
	return fileDescriptor_848ca6ed468fa3ce, []int{1}
}
func (m *EventExecScript) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventExecScript) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EventExecScript.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EventExecScript) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventExecScript.Merge(m, src)
}
func (m *EventExecScript) XXX_Size() int {
	return m.Size()
}
func (m *EventExecScript) XXX_DiscardUnknown() {
	xxx_messageInfo_EventExecScript.DiscardUnknown(m)
}

var xxx_messageInfo_EventExecScript proto.InternalMessageInfo

func (m *EventExecScript) GetRequest() *MsgExec {
	if m != nil {
		return m.Request
	}
	return nil
}

func (m *EventExecScript) GetResponse() *MsgExecResponse {
	if m != nil {
		return m.Response
	}
	return nil
}

// EventScriptEvent is an event emitted by the script itself.
type EventScriptEvent struct {
	// Address of the script
	Address string `protobuf:"bytes,1,opt,name=address,proto3" json:"address,omitempty"`
	// The key of the event.
	Key string `protobuf:"bytes,2,opt,name=key,proto3" json:"key,omitempty"`
	// The value of the event.
	Value string `protobuf:"bytes,3,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *EventScriptEvent) Reset()         { *m = EventScriptEvent{} }
func (m *EventScriptEvent) String() string { return proto.CompactTextString(m) }
func (*EventScriptEvent) ProtoMessage()    {}
func (*EventScriptEvent) Descriptor() ([]byte, []int) {
	return fileDescriptor_848ca6ed468fa3ce, []int{2}
}
func (m *EventScriptEvent) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventScriptEvent) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EventScriptEvent.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EventScriptEvent) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventScriptEvent.Merge(m, src)
}
func (m *EventScriptEvent) XXX_Size() int {
	return m.Size()
}
func (m *EventScriptEvent) XXX_DiscardUnknown() {
	xxx_messageInfo_EventScriptEvent.DiscardUnknown(m)
}

var xxx_messageInfo_EventScriptEvent proto.InternalMessageInfo

func (m *EventScriptEvent) GetAddress() string {
	if m != nil {
		return m.Address
	}
	return ""
}

func (m *EventScriptEvent) GetKey() string {
	if m != nil {
		return m.Key
	}
	return ""
}

func (m *EventScriptEvent) GetValue() string {
	if m != nil {
		return m.Value
	}
	return ""
}

// EventCreateNewScript is an event emitted when a new script is created.
type EventCreateNewScript struct {
	// The address of the newly created script.
	ScriptAddress string `protobuf:"bytes,1,opt,name=script_address,json=scriptAddress,proto3" json:"script_address,omitempty"`
	// The address of the creator.
	CreatorAddress string `protobuf:"bytes,2,opt,name=creator_address,json=creatorAddress,proto3" json:"creator_address,omitempty"`
	// The initial version of the script.
	Version uint64 `protobuf:"varint,3,opt,name=version,proto3" json:"version,omitempty"`
}

func (m *EventCreateNewScript) Reset()         { *m = EventCreateNewScript{} }
func (m *EventCreateNewScript) String() string { return proto.CompactTextString(m) }
func (*EventCreateNewScript) ProtoMessage()    {}
func (*EventCreateNewScript) Descriptor() ([]byte, []int) {
	return fileDescriptor_848ca6ed468fa3ce, []int{3}
}
func (m *EventCreateNewScript) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *EventCreateNewScript) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_EventCreateNewScript.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *EventCreateNewScript) XXX_Merge(src proto.Message) {
	xxx_messageInfo_EventCreateNewScript.Merge(m, src)
}
func (m *EventCreateNewScript) XXX_Size() int {
	return m.Size()
}
func (m *EventCreateNewScript) XXX_DiscardUnknown() {
	xxx_messageInfo_EventCreateNewScript.DiscardUnknown(m)
}

var xxx_messageInfo_EventCreateNewScript proto.InternalMessageInfo

func (m *EventCreateNewScript) GetScriptAddress() string {
	if m != nil {
		return m.ScriptAddress
	}
	return ""
}

func (m *EventCreateNewScript) GetCreatorAddress() string {
	if m != nil {
		return m.CreatorAddress
	}
	return ""
}

func (m *EventCreateNewScript) GetVersion() uint64 {
	if m != nil {
		return m.Version
	}
	return 0
}

func init() {
	proto.RegisterType((*EventUpdateScript)(nil), "dysonprotocol.script.v1.EventUpdateScript")
	proto.RegisterType((*EventExecScript)(nil), "dysonprotocol.script.v1.EventExecScript")
	proto.RegisterType((*EventScriptEvent)(nil), "dysonprotocol.script.v1.EventScriptEvent")
	proto.RegisterType((*EventCreateNewScript)(nil), "dysonprotocol.script.v1.EventCreateNewScript")
}

func init() {
	proto.RegisterFile("dysonprotocol/script/v1/events.proto", fileDescriptor_848ca6ed468fa3ce)
}

var fileDescriptor_848ca6ed468fa3ce = []byte{
	// 381 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x52, 0x3d, 0x6f, 0xe2, 0x40,
	0x10, 0xc5, 0xc7, 0xdd, 0x71, 0xec, 0xe9, 0xf8, 0xb0, 0x90, 0xce, 0x50, 0x58, 0xc8, 0xba, 0x53,
	0x68, 0xe2, 0x15, 0x49, 0x47, 0x97, 0x0f, 0xca, 0xa4, 0x70, 0x92, 0x26, 0x0d, 0x32, 0xf6, 0xc4,
	0x42, 0x01, 0xaf, 0xb3, 0xbb, 0x38, 0xb8, 0xcb, 0x4f, 0x88, 0x94, 0x3f, 0x95, 0x92, 0x32, 0x65,
	0x04, 0x7f, 0x24, 0xf2, 0xec, 0x1a, 0xc5, 0x05, 0x4a, 0xb3, 0x9a, 0x79, 0xf3, 0xde, 0xbe, 0x37,
	0xab, 0x25, 0xff, 0xc2, 0x4c, 0xb0, 0x38, 0xe1, 0x4c, 0xb2, 0x80, 0xcd, 0xa9, 0x08, 0xf8, 0x2c,
	0x91, 0x34, 0x1d, 0x52, 0x48, 0x21, 0x96, 0xc2, 0xc5, 0x89, 0xf9, 0xb7, 0xc4, 0x72, 0x15, 0xcb,
	0x4d, 0x87, 0xbd, 0x6e, 0xc0, 0xc4, 0x82, 0x89, 0x09, 0x8e, 0xa8, 0x6a, 0x94, 0xa6, 0xd7, 0xdf,
	0x77, 0xb3, 0x5c, 0x69, 0x46, 0x27, 0x62, 0x11, 0x53, 0xca, 0xbc, 0xd2, 0x68, 0xdb, 0x5f, 0xcc,
	0x62, 0x46, 0xf1, 0xd4, 0x50, 0x37, 0x62, 0x2c, 0x9a, 0x03, 0xc5, 0x6e, 0xba, 0xbc, 0xa3, 0x7e,
	0x9c, 0xa9, 0x91, 0x73, 0x48, 0xda, 0xe3, 0x3c, 0xe9, 0x4d, 0x12, 0xfa, 0x12, 0xae, 0xd0, 0xc4,
	0xb4, 0x48, 0x2d, 0x05, 0x2e, 0x66, 0x2c, 0xb6, 0x8c, 0xbe, 0x31, 0xf8, 0xee, 0x15, 0xad, 0xf3,
	0x62, 0x90, 0x26, 0xf2, 0xc7, 0x2b, 0x08, 0x34, 0x7b, 0x44, 0x6a, 0x1c, 0x1e, 0x96, 0x20, 0x24,
	0xb2, 0x7f, 0x1f, 0xf5, 0xdd, 0x3d, 0xeb, 0xba, 0x17, 0x22, 0xca, 0x85, 0x5e, 0x21, 0x30, 0xcf,
	0xc9, 0x2f, 0x0e, 0x22, 0x61, 0xb1, 0x00, 0xeb, 0x1b, 0x8a, 0x07, 0x5f, 0x8a, 0x35, 0xdf, 0xdb,
	0x29, 0x9d, 0x6b, 0xd2, 0xc2, 0x50, 0x2a, 0x10, 0x96, 0xf9, 0x0e, 0x7e, 0x18, 0x72, 0x10, 0x02,
	0x53, 0xd5, 0xbd, 0xa2, 0x35, 0x5b, 0xa4, 0x7a, 0x0f, 0x19, 0xda, 0xd5, 0xbd, 0xbc, 0x34, 0x3b,
	0xe4, 0x47, 0xea, 0xcf, 0x97, 0x60, 0x55, 0x11, 0x53, 0x8d, 0xf3, 0x64, 0x90, 0x0e, 0xde, 0x75,
	0xc6, 0xc1, 0x97, 0x70, 0x09, 0x8f, 0x7a, 0xe1, 0xff, 0xa4, 0xa1, 0x52, 0x4d, 0xca, 0x0e, 0x7f,
	0x14, 0x7a, 0xa2, 0x7d, 0x0e, 0x48, 0x33, 0xc8, 0x95, 0x8c, 0xef, 0x78, 0xca, 0xb3, 0xa1, 0xe1,
	0x82, 0xf8, 0xe9, 0xb9, 0xab, 0xa5, 0xe7, 0x3e, 0x1d, 0xbd, 0x6e, 0x6c, 0x63, 0xbd, 0xb1, 0x8d,
	0xf7, 0x8d, 0x6d, 0x3c, 0x6f, 0xed, 0xca, 0x7a, 0x6b, 0x57, 0xde, 0xb6, 0x76, 0xe5, 0xb6, 0xfc,
	0x3b, 0xdc, 0x80, 0x2d, 0xe8, 0xaa, 0xf8, 0x23, 0x32, 0x4b, 0x40, 0x4c, 0x7f, 0xe2, 0xf0, 0xf8,
	0x23, 0x00, 0x00, 0xff, 0xff, 0xce, 0x8d, 0x59, 0x94, 0xa2, 0x02, 0x00, 0x00,
}

func (m *EventUpdateScript) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventUpdateScript) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventUpdateScript) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Version != 0 {
		i = encodeVarintEvents(dAtA, i, uint64(m.Version))
		i--
		dAtA[i] = 0x8
	}
	return len(dAtA) - i, nil
}

func (m *EventExecScript) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventExecScript) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventExecScript) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Response != nil {
		{
			size, err := m.Response.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEvents(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0x12
	}
	if m.Request != nil {
		{
			size, err := m.Request.MarshalToSizedBuffer(dAtA[:i])
			if err != nil {
				return 0, err
			}
			i -= size
			i = encodeVarintEvents(dAtA, i, uint64(size))
		}
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *EventScriptEvent) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventScriptEvent) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventScriptEvent) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if len(m.Value) > 0 {
		i -= len(m.Value)
		copy(dAtA[i:], m.Value)
		i = encodeVarintEvents(dAtA, i, uint64(len(m.Value)))
		i--
		dAtA[i] = 0x1a
	}
	if len(m.Key) > 0 {
		i -= len(m.Key)
		copy(dAtA[i:], m.Key)
		i = encodeVarintEvents(dAtA, i, uint64(len(m.Key)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.Address) > 0 {
		i -= len(m.Address)
		copy(dAtA[i:], m.Address)
		i = encodeVarintEvents(dAtA, i, uint64(len(m.Address)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func (m *EventCreateNewScript) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *EventCreateNewScript) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *EventCreateNewScript) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.Version != 0 {
		i = encodeVarintEvents(dAtA, i, uint64(m.Version))
		i--
		dAtA[i] = 0x18
	}
	if len(m.CreatorAddress) > 0 {
		i -= len(m.CreatorAddress)
		copy(dAtA[i:], m.CreatorAddress)
		i = encodeVarintEvents(dAtA, i, uint64(len(m.CreatorAddress)))
		i--
		dAtA[i] = 0x12
	}
	if len(m.ScriptAddress) > 0 {
		i -= len(m.ScriptAddress)
		copy(dAtA[i:], m.ScriptAddress)
		i = encodeVarintEvents(dAtA, i, uint64(len(m.ScriptAddress)))
		i--
		dAtA[i] = 0xa
	}
	return len(dAtA) - i, nil
}

func encodeVarintEvents(dAtA []byte, offset int, v uint64) int {
	offset -= sovEvents(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *EventUpdateScript) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Version != 0 {
		n += 1 + sovEvents(uint64(m.Version))
	}
	return n
}

func (m *EventExecScript) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	if m.Request != nil {
		l = m.Request.Size()
		n += 1 + l + sovEvents(uint64(l))
	}
	if m.Response != nil {
		l = m.Response.Size()
		n += 1 + l + sovEvents(uint64(l))
	}
	return n
}

func (m *EventScriptEvent) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.Address)
	if l > 0 {
		n += 1 + l + sovEvents(uint64(l))
	}
	l = len(m.Key)
	if l > 0 {
		n += 1 + l + sovEvents(uint64(l))
	}
	l = len(m.Value)
	if l > 0 {
		n += 1 + l + sovEvents(uint64(l))
	}
	return n
}

func (m *EventCreateNewScript) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = len(m.ScriptAddress)
	if l > 0 {
		n += 1 + l + sovEvents(uint64(l))
	}
	l = len(m.CreatorAddress)
	if l > 0 {
		n += 1 + l + sovEvents(uint64(l))
	}
	if m.Version != 0 {
		n += 1 + sovEvents(uint64(m.Version))
	}
	return n
}

func sovEvents(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozEvents(x uint64) (n int) {
	return sovEvents(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *EventUpdateScript) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvents
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventUpdateScript: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventUpdateScript: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipEvents(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEvents
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *EventExecScript) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvents
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventExecScript: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventExecScript: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Request", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEvents
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEvents
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Request == nil {
				m.Request = &MsgExec{}
			}
			if err := m.Request.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Response", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				msglen |= int(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if msglen < 0 {
				return ErrInvalidLengthEvents
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthEvents
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if m.Response == nil {
				m.Response = &MsgExecResponse{}
			}
			if err := m.Response.Unmarshal(dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEvents(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEvents
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *EventScriptEvent) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvents
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventScriptEvent: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventScriptEvent: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Address", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvents
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvents
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Address = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Key", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvents
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvents
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Key = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Value", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvents
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvents
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Value = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		default:
			iNdEx = preIndex
			skippy, err := skipEvents(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEvents
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func (m *EventCreateNewScript) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowEvents
			}
			if iNdEx >= l {
				return io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= uint64(b&0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		fieldNum := int32(wire >> 3)
		wireType := int(wire & 0x7)
		if wireType == 4 {
			return fmt.Errorf("proto: EventCreateNewScript: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: EventCreateNewScript: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field ScriptAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvents
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvents
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.ScriptAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 2:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field CreatorAddress", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				stringLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			intStringLen := int(stringLen)
			if intStringLen < 0 {
				return ErrInvalidLengthEvents
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthEvents
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.CreatorAddress = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
		case 3:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field Version", wireType)
			}
			m.Version = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.Version |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipEvents(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthEvents
			}
			if (iNdEx + skippy) > l {
				return io.ErrUnexpectedEOF
			}
			iNdEx += skippy
		}
	}

	if iNdEx > l {
		return io.ErrUnexpectedEOF
	}
	return nil
}
func skipEvents(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowEvents
			}
			if iNdEx >= l {
				return 0, io.ErrUnexpectedEOF
			}
			b := dAtA[iNdEx]
			iNdEx++
			wire |= (uint64(b) & 0x7F) << shift
			if b < 0x80 {
				break
			}
		}
		wireType := int(wire & 0x7)
		switch wireType {
		case 0:
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				iNdEx++
				if dAtA[iNdEx-1] < 0x80 {
					break
				}
			}
		case 1:
			iNdEx += 8
		case 2:
			var length int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return 0, ErrIntOverflowEvents
				}
				if iNdEx >= l {
					return 0, io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				length |= (int(b) & 0x7F) << shift
				if b < 0x80 {
					break
				}
			}
			if length < 0 {
				return 0, ErrInvalidLengthEvents
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupEvents
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthEvents
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthEvents        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowEvents          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupEvents = fmt.Errorf("proto: unexpected end of group")
)
