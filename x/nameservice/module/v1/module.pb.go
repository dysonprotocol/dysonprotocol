// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: dysonprotocol/nameservice/module/v1/module.proto

package v1

import (
	_ "cosmossdk.io/depinject/appconfig/v1alpha1"
	fmt "fmt"
	_ "github.com/cosmos/cosmos-sdk/types/tx/amino"
	_ "github.com/cosmos/gogoproto/gogoproto"
	proto "github.com/cosmos/gogoproto/proto"
	_ "google.golang.org/protobuf/types/known/durationpb"
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

type Module struct {
	// max_name_length defines the maximum length of a name that can be registered
	MaxNameLength uint64 `protobuf:"varint,1,opt,name=max_name_length,json=maxNameLength,proto3" json:"max_name_length,omitempty" yaml:"max_name_length"`
	// max_metadata_length defines the maximum length of metadata that can be
	// stored
	MaxMetadataLength uint64 `protobuf:"varint,2,opt,name=max_metadata_length,json=maxMetadataLength,proto3" json:"max_metadata_length,omitempty" yaml:"max_metadata_length"`
	// authority defines the custom module authority. If not set, defaults to the
	// governance module.
	Authority string `protobuf:"bytes,3,opt,name=authority,proto3" json:"authority,omitempty"`
}

func (m *Module) Reset()         { *m = Module{} }
func (m *Module) String() string { return proto.CompactTextString(m) }
func (*Module) ProtoMessage()    {}
func (*Module) Descriptor() ([]byte, []int) {
	return fileDescriptor_1e2bb322a6a764e1, []int{0}
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

func (m *Module) GetMaxNameLength() uint64 {
	if m != nil {
		return m.MaxNameLength
	}
	return 0
}

func (m *Module) GetMaxMetadataLength() uint64 {
	if m != nil {
		return m.MaxMetadataLength
	}
	return 0
}

func (m *Module) GetAuthority() string {
	if m != nil {
		return m.Authority
	}
	return ""
}

func init() {
	proto.RegisterType((*Module)(nil), "dysonprotocol.nameservice.module.v1.Module")
}

func init() {
	proto.RegisterFile("dysonprotocol/nameservice/module/v1/module.proto", fileDescriptor_1e2bb322a6a764e1)
}

var fileDescriptor_1e2bb322a6a764e1 = []byte{
	// 324 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x91, 0xbf, 0x4e, 0xfb, 0x30,
	0x10, 0xc7, 0xeb, 0xdf, 0x0f, 0x55, 0xaa, 0x25, 0x84, 0x5a, 0x10, 0xaa, 0x22, 0xe4, 0x96, 0x30,
	0x50, 0x96, 0x9a, 0x8a, 0xad, 0x63, 0x59, 0x69, 0x87, 0x8e, 0x2c, 0xd5, 0x35, 0x31, 0x69, 0x24,
	0x3b, 0x17, 0x25, 0x4e, 0x94, 0xbe, 0x05, 0x13, 0x0f, 0xc3, 0xc4, 0xc8, 0xd8, 0x11, 0x31, 0x54,
	0xa8, 0x79, 0x83, 0x3e, 0x01, 0xaa, 0x31, 0xea, 0x9f, 0x85, 0xc5, 0xba, 0xfb, 0xfa, 0x73, 0x1f,
	0x59, 0x3e, 0x7a, 0xeb, 0xcf, 0x53, 0x8c, 0xe2, 0x04, 0x35, 0x7a, 0x28, 0x79, 0x04, 0x4a, 0xa4,
	0x22, 0xc9, 0x43, 0x4f, 0x70, 0x85, 0x7e, 0x26, 0x05, 0xcf, 0x7b, 0xb6, 0xea, 0x1a, 0xaa, 0x71,
	0xb5, 0x37, 0xd1, 0xdd, 0x99, 0xe8, 0x5a, 0x2e, 0xef, 0x39, 0x6d, 0x0f, 0x53, 0x85, 0x29, 0x87,
	0x38, 0xe6, 0x79, 0x0f, 0x64, 0x3c, 0x83, 0x7d, 0x8d, 0x73, 0x16, 0x60, 0x80, 0xa6, 0xe4, 0x9b,
	0xca, 0xa6, 0x2c, 0x40, 0x0c, 0xa4, 0xe0, 0xa6, 0x9b, 0x66, 0x4f, 0xdc, 0xcf, 0x12, 0xd0, 0x21,
	0x46, 0xf6, 0xbe, 0x0e, 0x2a, 0x8c, 0x90, 0x9b, 0xf3, 0x27, 0x72, 0x4b, 0x42, 0xab, 0x43, 0x63,
	0x6e, 0x0c, 0xe8, 0x89, 0x82, 0x62, 0xb2, 0x79, 0xd2, 0x44, 0x8a, 0x28, 0xd0, 0xb3, 0x26, 0x69,
	0x93, 0xce, 0xd1, 0xc0, 0x59, 0x2f, 0x5b, 0xe7, 0x73, 0x50, 0xb2, 0xef, 0x1e, 0x00, 0xee, 0xf8,
	0x58, 0x41, 0x31, 0x02, 0x25, 0x1e, 0x4c, 0xdf, 0x18, 0xd1, 0xd3, 0x0d, 0xa2, 0x84, 0x06, 0x1f,
	0x34, 0xfc, 0x7a, 0xfe, 0x19, 0x0f, 0x5b, 0x2f, 0x5b, 0xce, 0xd6, 0x73, 0x00, 0xb9, 0xe3, 0xba,
	0x82, 0x62, 0x68, 0x43, 0xeb, 0xbb, 0xa0, 0x35, 0xc8, 0xf4, 0x0c, 0x93, 0x50, 0xcf, 0x9b, 0xff,
	0xdb, 0xa4, 0x53, 0x1b, 0x6f, 0x83, 0xfe, 0xf5, 0xeb, 0xdb, 0xcb, 0x27, 0xb9, 0xa4, 0xad, 0xfd,
	0x4f, 0xf5, 0x50, 0xf1, 0x62, 0x77, 0x19, 0x83, 0xfb, 0xf7, 0x15, 0x23, 0x8b, 0x15, 0x23, 0x5f,
	0x2b, 0x46, 0x9e, 0x4b, 0x56, 0x59, 0x94, 0xac, 0xf2, 0x51, 0xb2, 0xca, 0xe3, 0xcd, 0x1f, 0xa3,
	0xdb, 0x3d, 0x4e, 0xab, 0x86, 0xba, 0xfb, 0x0e, 0x00, 0x00, 0xff, 0xff, 0x39, 0xe1, 0x3f, 0xdf,
	0xf5, 0x01, 0x00, 0x00,
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
	if len(m.Authority) > 0 {
		i -= len(m.Authority)
		copy(dAtA[i:], m.Authority)
		i = encodeVarintModule(dAtA, i, uint64(len(m.Authority)))
		i--
		dAtA[i] = 0x1a
	}
	if m.MaxMetadataLength != 0 {
		i = encodeVarintModule(dAtA, i, uint64(m.MaxMetadataLength))
		i--
		dAtA[i] = 0x10
	}
	if m.MaxNameLength != 0 {
		i = encodeVarintModule(dAtA, i, uint64(m.MaxNameLength))
		i--
		dAtA[i] = 0x8
	}
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
	if m.MaxNameLength != 0 {
		n += 1 + sovModule(uint64(m.MaxNameLength))
	}
	if m.MaxMetadataLength != 0 {
		n += 1 + sovModule(uint64(m.MaxMetadataLength))
	}
	l = len(m.Authority)
	if l > 0 {
		n += 1 + l + sovModule(uint64(l))
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
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxNameLength", wireType)
			}
			m.MaxNameLength = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowModule
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxNameLength |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 2:
			if wireType != 0 {
				return fmt.Errorf("proto: wrong wireType = %d for field MaxMetadataLength", wireType)
			}
			m.MaxMetadataLength = 0
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowModule
				}
				if iNdEx >= l {
					return io.ErrUnexpectedEOF
				}
				b := dAtA[iNdEx]
				iNdEx++
				m.MaxMetadataLength |= uint64(b&0x7F) << shift
				if b < 0x80 {
					break
				}
			}
		case 3:
			if wireType != 2 {
				return fmt.Errorf("proto: wrong wireType = %d for field Authority", wireType)
			}
			var stringLen uint64
			for shift := uint(0); ; shift += 7 {
				if shift >= 64 {
					return ErrIntOverflowModule
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
				return ErrInvalidLengthModule
			}
			postIndex := iNdEx + intStringLen
			if postIndex < 0 {
				return ErrInvalidLengthModule
			}
			if postIndex > l {
				return io.ErrUnexpectedEOF
			}
			m.Authority = string(dAtA[iNdEx:postIndex])
			iNdEx = postIndex
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
