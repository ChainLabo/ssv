// Code generated by protoc-gen-go. DO NOT EDIT.
// source: github.com/bloxapp/ssv/ibft/proto/msgs.proto

package proto

import (
	fmt "fmt"
	_ "github.com/gogo/protobuf/gogoproto"
	proto "github.com/golang/protobuf/proto"
	math "math"
)

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

// This is a compile-time assertion to ensure that this generated file
// is compatible with the proto package it is being compiled against.
// A compilation error at this line likely means your copy of the
// proto package needs to be updated.
const _ = proto.ProtoPackageIsVersion3 // please upgrade the proto package

type RoundState int32

const (
	RoundState_NotStarted  RoundState = 0
	RoundState_PrePrepare  RoundState = 1
	RoundState_Prepare     RoundState = 2
	RoundState_Commit      RoundState = 3
	RoundState_ChangeRound RoundState = 4
	RoundState_Decided     RoundState = 5
)

var RoundState_name = map[int32]string{
	0: "NotStarted",
	1: "PrePrepare",
	2: "Prepare",
	3: "Commit",
	4: "ChangeRound",
	5: "Decided",
}

var RoundState_value = map[string]int32{
	"NotStarted":  0,
	"PrePrepare":  1,
	"Prepare":     2,
	"Commit":      3,
	"ChangeRound": 4,
	"Decided":     5,
}

func (x RoundState) String() string {
	return proto.EnumName(RoundState_name, int32(x))
}

func (RoundState) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_dc1187fce89a5e11, []int{0}
}

type Message struct {
	Type                 RoundState `protobuf:"varint,1,opt,name=type,proto3,enum=proto.RoundState" json:"type,omitempty"`
	Round                uint64     `protobuf:"varint,2,opt,name=round,proto3" json:"round,omitempty"`
	Lambda               []byte     `protobuf:"bytes,3,opt,name=lambda,proto3" json:"lambda,omitempty"`
	PreviousLambda       []byte     `protobuf:"bytes,4,opt,name=previous_lambda,json=previousLambda,proto3" json:"previous_lambda,omitempty"`
	Value                []byte     `protobuf:"bytes,5,opt,name=value,proto3" json:"value,omitempty"`
	XXX_NoUnkeyedLiteral struct{}   `json:"-"`
	XXX_unrecognized     []byte     `json:"-"`
	XXX_sizecache        int32      `json:"-"`
}

func (m *Message) Reset()         { *m = Message{} }
func (m *Message) String() string { return proto.CompactTextString(m) }
func (*Message) ProtoMessage()    {}
func (*Message) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc1187fce89a5e11, []int{0}
}

func (m *Message) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Message.Unmarshal(m, b)
}
func (m *Message) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Message.Marshal(b, m, deterministic)
}
func (m *Message) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Message.Merge(m, src)
}
func (m *Message) XXX_Size() int {
	return xxx_messageInfo_Message.Size(m)
}
func (m *Message) XXX_DiscardUnknown() {
	xxx_messageInfo_Message.DiscardUnknown(m)
}

var xxx_messageInfo_Message proto.InternalMessageInfo

func (m *Message) GetType() RoundState {
	if m != nil {
		return m.Type
	}
	return RoundState_NotStarted
}

func (m *Message) GetRound() uint64 {
	if m != nil {
		return m.Round
	}
	return 0
}

func (m *Message) GetLambda() []byte {
	if m != nil {
		return m.Lambda
	}
	return nil
}

func (m *Message) GetPreviousLambda() []byte {
	if m != nil {
		return m.PreviousLambda
	}
	return nil
}

func (m *Message) GetValue() []byte {
	if m != nil {
		return m.Value
	}
	return nil
}

type SignedMessage struct {
	Message              *Message `protobuf:"bytes,1,opt,name=message,proto3" json:"message,omitempty"`
	Signature            []byte   `protobuf:"bytes,2,opt,name=signature,proto3" json:"signature,omitempty"`
	SignerIds            []uint64 `protobuf:"varint,3,rep,packed,name=signer_ids,json=signerIds,proto3" json:"signer_ids,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *SignedMessage) Reset()         { *m = SignedMessage{} }
func (m *SignedMessage) String() string { return proto.CompactTextString(m) }
func (*SignedMessage) ProtoMessage()    {}
func (*SignedMessage) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc1187fce89a5e11, []int{1}
}

func (m *SignedMessage) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_SignedMessage.Unmarshal(m, b)
}
func (m *SignedMessage) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_SignedMessage.Marshal(b, m, deterministic)
}
func (m *SignedMessage) XXX_Merge(src proto.Message) {
	xxx_messageInfo_SignedMessage.Merge(m, src)
}
func (m *SignedMessage) XXX_Size() int {
	return xxx_messageInfo_SignedMessage.Size(m)
}
func (m *SignedMessage) XXX_DiscardUnknown() {
	xxx_messageInfo_SignedMessage.DiscardUnknown(m)
}

var xxx_messageInfo_SignedMessage proto.InternalMessageInfo

func (m *SignedMessage) GetMessage() *Message {
	if m != nil {
		return m.Message
	}
	return nil
}

func (m *SignedMessage) GetSignature() []byte {
	if m != nil {
		return m.Signature
	}
	return nil
}

func (m *SignedMessage) GetSignerIds() []uint64 {
	if m != nil {
		return m.SignerIds
	}
	return nil
}

type ChangeRoundData struct {
	PreparedRound        uint64   `protobuf:"varint,1,opt,name=prepared_round,json=preparedRound,proto3" json:"prepared_round,omitempty"`
	PreparedValue        []byte   `protobuf:"bytes,2,opt,name=prepared_value,json=preparedValue,proto3" json:"prepared_value,omitempty"`
	JustificationMsg     *Message `protobuf:"bytes,3,opt,name=justification_msg,json=justificationMsg,proto3" json:"justification_msg,omitempty"`
	JustificationSig     []byte   `protobuf:"bytes,4,opt,name=justification_sig,json=justificationSig,proto3" json:"justification_sig,omitempty"`
	SignerIds            []uint64 `protobuf:"varint,5,rep,packed,name=signer_ids,json=signerIds,proto3" json:"signer_ids,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *ChangeRoundData) Reset()         { *m = ChangeRoundData{} }
func (m *ChangeRoundData) String() string { return proto.CompactTextString(m) }
func (*ChangeRoundData) ProtoMessage()    {}
func (*ChangeRoundData) Descriptor() ([]byte, []int) {
	return fileDescriptor_dc1187fce89a5e11, []int{2}
}

func (m *ChangeRoundData) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_ChangeRoundData.Unmarshal(m, b)
}
func (m *ChangeRoundData) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_ChangeRoundData.Marshal(b, m, deterministic)
}
func (m *ChangeRoundData) XXX_Merge(src proto.Message) {
	xxx_messageInfo_ChangeRoundData.Merge(m, src)
}
func (m *ChangeRoundData) XXX_Size() int {
	return xxx_messageInfo_ChangeRoundData.Size(m)
}
func (m *ChangeRoundData) XXX_DiscardUnknown() {
	xxx_messageInfo_ChangeRoundData.DiscardUnknown(m)
}

var xxx_messageInfo_ChangeRoundData proto.InternalMessageInfo

func (m *ChangeRoundData) GetPreparedRound() uint64 {
	if m != nil {
		return m.PreparedRound
	}
	return 0
}

func (m *ChangeRoundData) GetPreparedValue() []byte {
	if m != nil {
		return m.PreparedValue
	}
	return nil
}

func (m *ChangeRoundData) GetJustificationMsg() *Message {
	if m != nil {
		return m.JustificationMsg
	}
	return nil
}

func (m *ChangeRoundData) GetJustificationSig() []byte {
	if m != nil {
		return m.JustificationSig
	}
	return nil
}

func (m *ChangeRoundData) GetSignerIds() []uint64 {
	if m != nil {
		return m.SignerIds
	}
	return nil
}

func init() {
	proto.RegisterEnum("proto.RoundState", RoundState_name, RoundState_value)
	proto.RegisterType((*Message)(nil), "proto.Message")
	proto.RegisterType((*SignedMessage)(nil), "proto.SignedMessage")
	proto.RegisterType((*ChangeRoundData)(nil), "proto.ChangeRoundData")
}

func init() {
	proto.RegisterFile("github.com/bloxapp/ssv/ibft/proto/msgs.proto", fileDescriptor_dc1187fce89a5e11)
}

var fileDescriptor_dc1187fce89a5e11 = []byte{
	// 444 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x84, 0x92, 0xdf, 0x6e, 0xd3, 0x30,
	0x14, 0xc6, 0xe7, 0x35, 0x69, 0xc5, 0xe9, 0xd6, 0x66, 0xd6, 0x84, 0x22, 0x24, 0x44, 0x29, 0x9a,
	0xa8, 0xf8, 0xd3, 0x88, 0xf1, 0x04, 0x6c, 0xbb, 0x41, 0x62, 0x68, 0x4a, 0x25, 0x2e, 0xb8, 0xa9,
	0x9c, 0xf8, 0xd4, 0x33, 0x6a, 0xe2, 0xc8, 0x76, 0x2a, 0xb8, 0xe5, 0x3d, 0x78, 0x1f, 0x9e, 0x82,
	0x07, 0xe1, 0x0a, 0xc5, 0x4e, 0x58, 0xa8, 0x40, 0x5c, 0xe5, 0x7c, 0xe7, 0xfc, 0x4e, 0x74, 0xbe,
	0x4f, 0x86, 0x17, 0x42, 0xda, 0xdb, 0x3a, 0x5b, 0xe6, 0xaa, 0x48, 0xb2, 0xad, 0xfa, 0xcc, 0xaa,
	0x2a, 0x31, 0x66, 0x97, 0xc8, 0x6c, 0x63, 0x93, 0x4a, 0x2b, 0xab, 0x92, 0xc2, 0x08, 0xb3, 0x74,
	0x25, 0x0d, 0xdd, 0xe7, 0xc1, 0xcb, 0xde, 0x92, 0x50, 0x42, 0x79, 0x30, 0xab, 0x37, 0x4e, 0xf9,
	0xad, 0xa6, 0xf2, 0x5b, 0xf3, 0x6f, 0x04, 0x46, 0xd7, 0x68, 0x0c, 0x13, 0x48, 0xcf, 0x20, 0xb0,
	0x5f, 0x2a, 0x8c, 0xc9, 0x8c, 0x2c, 0x26, 0xe7, 0x27, 0x9e, 0x58, 0xa6, 0xaa, 0x2e, 0xf9, 0xca,
	0x32, 0x8b, 0xa9, 0x1b, 0xd3, 0x53, 0x08, 0x75, 0xd3, 0x8b, 0x0f, 0x67, 0x64, 0x11, 0xa4, 0x5e,
	0xd0, 0xfb, 0x30, 0xdc, 0xb2, 0x22, 0xe3, 0x2c, 0x1e, 0xcc, 0xc8, 0xe2, 0x28, 0x6d, 0x15, 0x7d,
	0x0a, 0xd3, 0x4a, 0xe3, 0x4e, 0xaa, 0xda, 0xac, 0x5b, 0x20, 0x70, 0xc0, 0xa4, 0x6b, 0xbf, 0xf3,
	0xe0, 0x29, 0x84, 0x3b, 0xb6, 0xad, 0x31, 0x0e, 0xdd, 0xd8, 0x8b, 0xf9, 0x57, 0x02, 0xc7, 0x2b,
	0x29, 0x4a, 0xe4, 0xdd, 0x95, 0x4b, 0x18, 0x15, 0xbe, 0x74, 0x87, 0x8e, 0xcf, 0x27, 0xed, 0xa1,
	0x2d, 0x70, 0x11, 0x7c, 0xff, 0xf1, 0xe8, 0x20, 0xed, 0x20, 0x3a, 0x87, 0x7b, 0x46, 0x8a, 0x92,
	0xd9, 0x5a, 0xa3, 0x3b, 0xf9, 0xa8, 0x25, 0xee, 0xda, 0xf4, 0x21, 0x40, 0x23, 0x50, 0xaf, 0x25,
	0x37, 0xf1, 0x60, 0x36, 0x58, 0x04, 0x7e, 0x8c, 0xfa, 0x2d, 0x37, 0xf3, 0x9f, 0x04, 0xa6, 0x97,
	0xb7, 0xac, 0x14, 0xe8, 0xc2, 0xb8, 0x62, 0x96, 0xd1, 0x33, 0x68, 0x0c, 0x54, 0x4c, 0x23, 0x5f,
	0xfb, 0x38, 0x88, 0x8b, 0xe3, 0xb8, 0xeb, 0x3a, 0x94, 0x3e, 0xef, 0x61, 0xde, 0x5e, 0xff, 0x84,
	0xdf, 0xf0, 0x87, 0x66, 0x44, 0xdf, 0xc0, 0xc9, 0xa7, 0xda, 0x58, 0xb9, 0x91, 0x39, 0xb3, 0x52,
	0x95, 0xeb, 0xc2, 0x08, 0x17, 0xe7, 0xbf, 0x4c, 0x46, 0x7f, 0xe0, 0xd7, 0x46, 0xd0, 0x57, 0xfb,
	0xbf, 0x30, 0x52, 0xf8, 0xc0, 0xff, 0xba, 0xb2, 0x92, 0x62, 0xcf, 0x7c, 0xb8, 0x67, 0xfe, 0x59,
	0x0e, 0x70, 0xf7, 0x04, 0xe8, 0x04, 0xe0, 0xbd, 0xb2, 0x2b, 0xcb, 0xb4, 0x45, 0x1e, 0x1d, 0x34,
	0xfa, 0x46, 0xe3, 0x8d, 0xb7, 0x11, 0x11, 0x3a, 0x86, 0x51, 0x27, 0x0e, 0x29, 0xc0, 0xf0, 0x52,
	0x15, 0x85, 0xb4, 0xd1, 0x80, 0x4e, 0x61, 0xdc, 0x8b, 0x30, 0x0a, 0x1a, 0xf2, 0x0a, 0x73, 0xc9,
	0x91, 0x47, 0xe1, 0xc5, 0x93, 0x8f, 0x8f, 0xff, 0xfb, 0xd8, 0xb3, 0xa1, 0xfb, 0xbc, 0xfe, 0x15,
	0x00, 0x00, 0xff, 0xff, 0xde, 0x89, 0x95, 0x81, 0x18, 0x03, 0x00, 0x00,
}
