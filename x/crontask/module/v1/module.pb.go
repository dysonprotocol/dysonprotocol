// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: dysonprotocol/crontask/module/v1/module.proto

package v1

import (
	_ "cosmossdk.io/depinject/appconfig/v1alpha1"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	github_com_cosmos_gogoproto_types "github.com/cosmos/gogoproto/types"
	_ "google.golang.org/protobuf/types/known/durationpb"
	io "io"
	math "math"
	math_bits "math/bits"
	time "time"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf
var _ = time.Kitchen

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.GoGoProtoPackageIsVersion3 // please upgrade the proto package

// Module is the config object of the crontask module.
type Module struct {
	// max_execution_period defines the max duration after a cron task is
	// scheduled that it can be executed.
	MaxExecutionPeriod time.Duration `protobuf:"bytes,1,opt,name=max_execution_period,json=maxExecutionPeriod,proto3,stdduration" json:"max_execution_period"`
	// MaxMetadataLen defines the max chars allowed in metadata field
	// Defaults to 255 if not explicitly set.
	MaxMetadataLen uint64 `protobuf:"varint,2,opt,name=max_metadata_len,json=maxMetadataLen,proto3" json:"max_metadata_len,omitempty"`
}

func (m *Module) Reset()         { *m = Module{} }
func (m *Module) String() string { return proto.CompactTextString(m) }
func (*Module) ProtoMessage()    {}
func (*Module) Descriptor() ([]byte, []int) {
	return fileDescriptor_96f90fa30acffc71, []int{0}
}
func (m *Module) XXX_Unmarshal(b []byte) error {
	return m.Unmarshal(b)
}
func (m *Module) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	if deterministic {
		return xxx_messageInfo_Module.Marshal(b, m, deterministic)
	} else {
		b = b[:cap(b)]
		n, err := m.MarshalToSizedBuffer(b)
		if err != nil {
			return nil, err
		}
		return b[:n], nil
	}
}
func (m *Module) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Module.Merge(m, src)
}
func (m *Module) XXX_Size() int {
	return m.Size()
}
func (m *Module) XXX_DiscardUnknown() {
	xxx_messageInfo_Module.DiscardUnknown(m)
}

var xxx_messageInfo_Module proto.InternalMessageInfo

func (m *Module) GetMaxExecutionPeriod() time.Duration {
	if m != nil {
		return m.MaxExecutionPeriod
	}
	return 0
}

func (m *Module) GetMaxMetadataLen() uint64 {
	if m != nil {
		return m.MaxMetadataLen
	}
	return 0
}

func init() {
	proto.RegisterType((*Module)(nil), "dysonprotocol.crontask.module.v1.Module")
}

func init() {
	proto.RegisterFile("dysonprotocol/crontask/module/v1/module.proto", fileDescriptor_96f90fa30acffc71)
}

var fileDescriptor_96f90fa30acffc71 = []byte{
	// 313 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xe2, 0xd2, 0x4d, 0xa9, 0x2c, 0xce,
	0xcf, 0x2b, 0x28, 0xca, 0x2f, 0xc9, 0x4f, 0xce, 0xcf, 0xd1, 0x4f, 0x2e, 0xca, 0xcf, 0x2b, 0x49,
	0x2c, 0xce, 0xd6, 0xcf, 0xcd, 0x4f, 0x29, 0xcd, 0x49, 0xd5, 0x2f, 0x33, 0x84, 0xb2, 0xf4, 0xc0,
	0x4a, 0x84, 0x14, 0x50, 0x94, 0xeb, 0xc1, 0x94, 0xeb, 0x41, 0x15, 0x95, 0x19, 0x4a, 0x29, 0x24,
	0xe7, 0x17, 0xe7, 0xe6, 0x17, 0xeb, 0x27, 0x16, 0x14, 0xe8, 0x97, 0x19, 0x26, 0xe6, 0x14, 0x64,
	0x24, 0xa2, 0x9a, 0x21, 0x25, 0x92, 0x9e, 0x9f, 0x9e, 0x0f, 0x66, 0xea, 0x83, 0x58, 0x50, 0x51,
	0xb9, 0xf4, 0xfc, 0xfc, 0xf4, 0x9c, 0x54, 0x7d, 0x30, 0x2f, 0xa9, 0x34, 0x4d, 0x3f, 0xa5, 0xb4,
	0x28, 0xb1, 0x24, 0x33, 0x3f, 0x0f, 0x2a, 0x2f, 0x98, 0x98, 0x9b, 0x99, 0x97, 0xaf, 0x0f, 0x26,
	0x21, 0x42, 0x4a, 0x5b, 0x18, 0xb9, 0xd8, 0x7c, 0xc1, 0x26, 0x0b, 0x45, 0x71, 0x89, 0xe4, 0x26,
	0x56, 0xc4, 0xa7, 0x56, 0xa4, 0x26, 0x97, 0x82, 0x34, 0xc5, 0x17, 0xa4, 0x16, 0x65, 0xe6, 0xa7,
	0x48, 0x30, 0x2a, 0x30, 0x6a, 0x70, 0x1b, 0x49, 0xea, 0x41, 0x0c, 0xd7, 0x83, 0x19, 0xae, 0xe7,
	0x02, 0x35, 0xdc, 0x89, 0xf7, 0xc4, 0x3d, 0x79, 0x86, 0x19, 0xf7, 0xe5, 0x19, 0x57, 0x3c, 0xdf,
	0xa0, 0xc5, 0x18, 0x24, 0x94, 0x9b, 0x58, 0xe1, 0x0a, 0x33, 0x24, 0x00, 0x6c, 0x86, 0x90, 0x06,
	0x97, 0x00, 0xc8, 0xec, 0xdc, 0xd4, 0x92, 0xc4, 0x94, 0xc4, 0x92, 0xc4, 0xf8, 0x9c, 0xd4, 0x3c,
	0x09, 0x26, 0x05, 0x46, 0x0d, 0x96, 0x20, 0xbe, 0xdc, 0xc4, 0x0a, 0x5f, 0xa8, 0xb0, 0x4f, 0x6a,
	0x9e, 0x95, 0xca, 0xae, 0x03, 0xd3, 0x6e, 0x31, 0xca, 0x71, 0xc9, 0xa0, 0x85, 0x52, 0x7e, 0xae,
	0x7e, 0x05, 0x3c, 0x68, 0x9d, 0x1c, 0x4e, 0x3c, 0x92, 0x63, 0xbc, 0xf0, 0x48, 0x8e, 0xf1, 0xc1,
	0x23, 0x39, 0xc6, 0x09, 0x8f, 0xe5, 0x18, 0x2e, 0x3c, 0x96, 0x63, 0xb8, 0xf1, 0x58, 0x8e, 0x21,
	0x4a, 0x0d, 0x9f, 0x3e, 0x44, 0x94, 0x24, 0xb1, 0x81, 0x95, 0x18, 0x03, 0x02, 0x00, 0x00, 0xff,
	0xff, 0x82, 0x3d, 0x8a, 0x32, 0xbd, 0x01, 0x00, 0x00,
}

func (m *Module) Marshal() (dAtA []byte, err error) {
	size := m.Size()
	dAtA = make([]byte, size)
	n, err := m.MarshalToSizedBuffer(dAtA[:size])
	if err != nil {
		return nil, err
	}
	return dAtA[:n], nil
}

func (m *Module) MarshalTo(dAtA []byte) (int, error) {
	size := m.Size()
	return m.MarshalToSizedBuffer(dAtA[:size])
}

func (m *Module) MarshalToSizedBuffer(dAtA []byte) (int, error) {
	i := len(dAtA)
	_ = i
	var l int
	_ = l
	if m.MaxMetadataLen != 0 {
		i = encodeVarintModule(dAtA, i, uint64(m.MaxMetadataLen))
		i--
		dAtA[i] = 0x10
	}
	n1, err1 := github_com_cosmos_gogoproto_types.StdDurationMarshalTo(m.MaxExecutionPeriod, dAtA[i-github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.MaxExecutionPeriod):])
	if err1 != nil {
		return 0, err1
	}
	i -= n1
	i = encodeVarintModule(dAtA, i, uint64(n1))
	i--
	dAtA[i] = 0xa
	return len(dAtA) - i, nil
}

func encodeVarintModule(dAtA []byte, offset int, v uint64) int {
	offset -= sovModule(v)
	base := offset
	for v >= 1<<7 {
		dAtA[offset] = uint8(v&0x7f | 0x80)
		v >>= 7
		offset++
	}
	dAtA[offset] = uint8(v)
	return base
}
func (m *Module) Size() (n int) {
	if m == nil {
		return 0
	}
	var l int
	_ = l
	l = github_com_cosmos_gogoproto_types.SizeOfStdDuration(m.MaxExecutionPeriod)
	n += 1 + l + sovModule(uint64(l))
	if m.MaxMetadataLen != 0 {
		n += 1 + sovModule(uint64(m.MaxMetadataLen))
	}
	return n
}

func sovModule(x uint64) (n int) {
	return (math_bits.Len64(x|1) + 6) / 7
}
func sozModule(x uint64) (n int) {
	return sovModule(uint64((x << 1) ^ uint64((int64(x) >> 63))))
}
func (m *Module) Unmarshal(dAtA []byte) error {
	l := len(dAtA)
	iNdEx := 0
	for iNdEx < l {
		preIndex := iNdEx
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return ErrIntOverflowModule
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
			return fmt.Errorf("proto: Module: wiretype end group for non-group")
		}
		if fieldNum <= 0 {
			return fmt.Errorf("proto: Module: illegal tag %d (wire type %d)", fieldNum, wire)
		}
		switch fieldNum {
		case 1:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxExecutionPeriod", wireType)
			}
			var msglen int
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowModule
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
				return ErrInvalidLengthModule
			}
			postIndex := iNdEx + msglen
			if postIndex < 0 {
				return ErrInvalidLengthModule
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			if err := github_com_cosmos_gogoproto_types.StdDurationUnmarshal(&m.MaxExecutionPeriod, dAtA[iNdEx:postIndex]); err != nil {
				return err
			}
			iNdEx = postIndex
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxMetadataLen", wireType)
			}
			m.MaxMetadataLen = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowModule
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxMetadataLen |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		default:
			iNdEx = preIndex
			skippy, err := skipModule(dAtA[iNdEx:])
			if err != nil {
				return err
			}
			if (skippy < 0) || (iNdEx+skippy) < 0 {
				return ErrInvalidLengthModule
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
func skipModule(dAtA []byte) (n int, err error) {
	l := len(dAtA)
	iNdEx := 0
	depth := 0
	for iNdEx < l {
		var wire uint64
		for shift := uint(0); ; shift += 7 {
			if shift >= 64 {
				return 0, ErrIntOverflowModule
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
					return 0, ErrIntOverflowModule
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
					return 0, ErrIntOverflowModule
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
				return 0, ErrInvalidLengthModule
			}
			iNdEx += length
		case 3:
			depth++
		case 4:
			if depth == 0 {
				return 0, ErrUnexpectedEndOfGroupModule
			}
			depth--
		case 5:
			iNdEx += 4
		default:
			return 0, fmt.Errorf("proto: illegal wireType %d", wireType)
		}
		if iNdEx < 0 {
			return 0, ErrInvalidLengthModule
		}
		if depth == 0 {
			return iNdEx, nil
		}
	}
	return 0, io.ErrUnexpectedEOF
}

var (
	ErrInvalidLengthModule        = fmt.Errorf("proto: negative length found during unmarshaling")
	ErrIntOverflowModule          = fmt.Errorf("proto: integer overflow")
	ErrUnexpectedEndOfGroupModule = fmt.Errorf("proto: unexpected end of group")
)
