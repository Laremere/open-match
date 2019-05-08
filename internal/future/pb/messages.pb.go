// Code generated by protoc-gen-go. DO NOT EDIT.
// source: api/messages.proto

package pb

import (
	fmt "fmt"
	proto "github.com/golang/protobuf/proto"
	any "github.com/golang/protobuf/ptypes/any"
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

type Ticket struct {
	Id                   string             `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Properties           *any.Any           `protobuf:"bytes,2,opt,name=properties,proto3" json:"properties,omitempty"`
	Dargs                map[string]float64 `protobuf:"bytes,3,rep,name=dargs,proto3" json:"dargs,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"fixed64,2,opt,name=value,proto3"`
	XXX_NoUnkeyedLiteral struct{}           `json:"-"`
	XXX_unrecognized     []byte             `json:"-"`
	XXX_sizecache        int32              `json:"-"`
}

func (m *Ticket) Reset()         { *m = Ticket{} }
func (m *Ticket) String() string { return proto.CompactTextString(m) }
func (*Ticket) ProtoMessage()    {}
func (*Ticket) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{0}
}

func (m *Ticket) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Ticket.Unmarshal(m, b)
}
func (m *Ticket) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Ticket.Marshal(b, m, deterministic)
}
func (m *Ticket) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Ticket.Merge(m, src)
}
func (m *Ticket) XXX_Size() int {
	return xxx_messageInfo_Ticket.Size(m)
}
func (m *Ticket) XXX_DiscardUnknown() {
	xxx_messageInfo_Ticket.DiscardUnknown(m)
}

var xxx_messageInfo_Ticket proto.InternalMessageInfo

func (m *Ticket) GetId() string {
	if m != nil {
		return m.Id
	}
	return ""
}

func (m *Ticket) GetProperties() *any.Any {
	if m != nil {
		return m.Properties
	}
	return nil
}

func (m *Ticket) GetDargs() map[string]float64 {
	if m != nil {
		return m.Dargs
	}
	return nil
}

type Assignment struct {
	Connection           string   `protobuf:"bytes,1,opt,name=connection,proto3" json:"connection,omitempty"`
	Properties           *any.Any `protobuf:"bytes,2,opt,name=properties,proto3" json:"properties,omitempty"`
	Error                string   `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Assignment) Reset()         { *m = Assignment{} }
func (m *Assignment) String() string { return proto.CompactTextString(m) }
func (*Assignment) ProtoMessage()    {}
func (*Assignment) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{1}
}

func (m *Assignment) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Assignment.Unmarshal(m, b)
}
func (m *Assignment) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Assignment.Marshal(b, m, deterministic)
}
func (m *Assignment) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Assignment.Merge(m, src)
}
func (m *Assignment) XXX_Size() int {
	return xxx_messageInfo_Assignment.Size(m)
}
func (m *Assignment) XXX_DiscardUnknown() {
	xxx_messageInfo_Assignment.DiscardUnknown(m)
}

var xxx_messageInfo_Assignment proto.InternalMessageInfo

func (m *Assignment) GetConnection() string {
	if m != nil {
		return m.Connection
	}
	return ""
}

func (m *Assignment) GetProperties() *any.Any {
	if m != nil {
		return m.Properties
	}
	return nil
}

func (m *Assignment) GetError() string {
	if m != nil {
		return m.Error
	}
	return ""
}

type Index struct {
	// Types that are valid to be assigned to Value:
	//	*Index_PoolIndex
	//	*Index_RangeIndex
	Value                isIndex_Value `protobuf_oneof:"value"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *Index) Reset()         { *m = Index{} }
func (m *Index) String() string { return proto.CompactTextString(m) }
func (*Index) ProtoMessage()    {}
func (*Index) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{2}
}

func (m *Index) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Index.Unmarshal(m, b)
}
func (m *Index) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Index.Marshal(b, m, deterministic)
}
func (m *Index) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Index.Merge(m, src)
}
func (m *Index) XXX_Size() int {
	return xxx_messageInfo_Index.Size(m)
}
func (m *Index) XXX_DiscardUnknown() {
	xxx_messageInfo_Index.DiscardUnknown(m)
}

var xxx_messageInfo_Index proto.InternalMessageInfo

type isIndex_Value interface {
	isIndex_Value()
}

type Index_PoolIndex struct {
	PoolIndex *PoolIndex `protobuf:"bytes,1,opt,name=pool_index,json=poolIndex,proto3,oneof"`
}

type Index_RangeIndex struct {
	RangeIndex *RangeIndex `protobuf:"bytes,2,opt,name=range_index,json=rangeIndex,proto3,oneof"`
}

func (*Index_PoolIndex) isIndex_Value() {}

func (*Index_RangeIndex) isIndex_Value() {}

func (m *Index) GetValue() isIndex_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Index) GetPoolIndex() *PoolIndex {
	if x, ok := m.GetValue().(*Index_PoolIndex); ok {
		return x.PoolIndex
	}
	return nil
}

func (m *Index) GetRangeIndex() *RangeIndex {
	if x, ok := m.GetValue().(*Index_RangeIndex); ok {
		return x.RangeIndex
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Index) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Index_PoolIndex)(nil),
		(*Index_RangeIndex)(nil),
	}
}

type PoolIndex struct {
	Index                *Index    `protobuf:"bytes,1,opt,name=index,proto3" json:"index,omitempty"`
	Filters              []*Filter `protobuf:"bytes,2,rep,name=filters,proto3" json:"filters,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *PoolIndex) Reset()         { *m = PoolIndex{} }
func (m *PoolIndex) String() string { return proto.CompactTextString(m) }
func (*PoolIndex) ProtoMessage()    {}
func (*PoolIndex) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{3}
}

func (m *PoolIndex) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PoolIndex.Unmarshal(m, b)
}
func (m *PoolIndex) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PoolIndex.Marshal(b, m, deterministic)
}
func (m *PoolIndex) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PoolIndex.Merge(m, src)
}
func (m *PoolIndex) XXX_Size() int {
	return xxx_messageInfo_PoolIndex.Size(m)
}
func (m *PoolIndex) XXX_DiscardUnknown() {
	xxx_messageInfo_PoolIndex.DiscardUnknown(m)
}

var xxx_messageInfo_PoolIndex proto.InternalMessageInfo

func (m *PoolIndex) GetIndex() *Index {
	if m != nil {
		return m.Index
	}
	return nil
}

func (m *PoolIndex) GetFilters() []*Filter {
	if m != nil {
		return m.Filters
	}
	return nil
}

type RangeIndex struct {
	Darg                 string   `protobuf:"bytes,1,opt,name=darg,proto3" json:"darg,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RangeIndex) Reset()         { *m = RangeIndex{} }
func (m *RangeIndex) String() string { return proto.CompactTextString(m) }
func (*RangeIndex) ProtoMessage()    {}
func (*RangeIndex) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{4}
}

func (m *RangeIndex) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RangeIndex.Unmarshal(m, b)
}
func (m *RangeIndex) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RangeIndex.Marshal(b, m, deterministic)
}
func (m *RangeIndex) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RangeIndex.Merge(m, src)
}
func (m *RangeIndex) XXX_Size() int {
	return xxx_messageInfo_RangeIndex.Size(m)
}
func (m *RangeIndex) XXX_DiscardUnknown() {
	xxx_messageInfo_RangeIndex.DiscardUnknown(m)
}

var xxx_messageInfo_RangeIndex proto.InternalMessageInfo

func (m *RangeIndex) GetDarg() string {
	if m != nil {
		return m.Darg
	}
	return ""
}

type Filter struct {
	// Types that are valid to be assigned to Value:
	//	*Filter_RangeFilter
	Value                isFilter_Value `protobuf_oneof:"value"`
	XXX_NoUnkeyedLiteral struct{}       `json:"-"`
	XXX_unrecognized     []byte         `json:"-"`
	XXX_sizecache        int32          `json:"-"`
}

func (m *Filter) Reset()         { *m = Filter{} }
func (m *Filter) String() string { return proto.CompactTextString(m) }
func (*Filter) ProtoMessage()    {}
func (*Filter) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{5}
}

func (m *Filter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Filter.Unmarshal(m, b)
}
func (m *Filter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Filter.Marshal(b, m, deterministic)
}
func (m *Filter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Filter.Merge(m, src)
}
func (m *Filter) XXX_Size() int {
	return xxx_messageInfo_Filter.Size(m)
}
func (m *Filter) XXX_DiscardUnknown() {
	xxx_messageInfo_Filter.DiscardUnknown(m)
}

var xxx_messageInfo_Filter proto.InternalMessageInfo

type isFilter_Value interface {
	isFilter_Value()
}

type Filter_RangeFilter struct {
	RangeFilter *RangeFilter `protobuf:"bytes,1,opt,name=range_filter,json=rangeFilter,proto3,oneof"`
}

func (*Filter_RangeFilter) isFilter_Value() {}

func (m *Filter) GetValue() isFilter_Value {
	if m != nil {
		return m.Value
	}
	return nil
}

func (m *Filter) GetRangeFilter() *RangeFilter {
	if x, ok := m.GetValue().(*Filter_RangeFilter); ok {
		return x.RangeFilter
	}
	return nil
}

// XXX_OneofWrappers is for the internal use of the proto package.
func (*Filter) XXX_OneofWrappers() []interface{} {
	return []interface{}{
		(*Filter_RangeFilter)(nil),
	}
}

type RangeFilter struct {
	Darg                 string   `protobuf:"bytes,1,opt,name=darg,proto3" json:"darg,omitempty"`
	Min                  float64  `protobuf:"fixed64,2,opt,name=min,proto3" json:"min,omitempty"`
	Max                  float64  `protobuf:"fixed64,3,opt,name=max,proto3" json:"max,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *RangeFilter) Reset()         { *m = RangeFilter{} }
func (m *RangeFilter) String() string { return proto.CompactTextString(m) }
func (*RangeFilter) ProtoMessage()    {}
func (*RangeFilter) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{6}
}

func (m *RangeFilter) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_RangeFilter.Unmarshal(m, b)
}
func (m *RangeFilter) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_RangeFilter.Marshal(b, m, deterministic)
}
func (m *RangeFilter) XXX_Merge(src proto.Message) {
	xxx_messageInfo_RangeFilter.Merge(m, src)
}
func (m *RangeFilter) XXX_Size() int {
	return xxx_messageInfo_RangeFilter.Size(m)
}
func (m *RangeFilter) XXX_DiscardUnknown() {
	xxx_messageInfo_RangeFilter.DiscardUnknown(m)
}

var xxx_messageInfo_RangeFilter proto.InternalMessageInfo

func (m *RangeFilter) GetDarg() string {
	if m != nil {
		return m.Darg
	}
	return ""
}

func (m *RangeFilter) GetMin() float64 {
	if m != nil {
		return m.Min
	}
	return 0
}

func (m *RangeFilter) GetMax() float64 {
	if m != nil {
		return m.Max
	}
	return 0
}

type Query struct {
	Index                *Index    `protobuf:"bytes,1,opt,name=index,proto3" json:"index,omitempty"`
	Filters              []*Filter `protobuf:"bytes,2,rep,name=filters,proto3" json:"filters,omitempty"`
	XXX_NoUnkeyedLiteral struct{}  `json:"-"`
	XXX_unrecognized     []byte    `json:"-"`
	XXX_sizecache        int32     `json:"-"`
}

func (m *Query) Reset()         { *m = Query{} }
func (m *Query) String() string { return proto.CompactTextString(m) }
func (*Query) ProtoMessage()    {}
func (*Query) Descriptor() ([]byte, []int) {
	return fileDescriptor_cb9fb1f207fd5b8c, []int{7}
}

func (m *Query) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Query.Unmarshal(m, b)
}
func (m *Query) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Query.Marshal(b, m, deterministic)
}
func (m *Query) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Query.Merge(m, src)
}
func (m *Query) XXX_Size() int {
	return xxx_messageInfo_Query.Size(m)
}
func (m *Query) XXX_DiscardUnknown() {
	xxx_messageInfo_Query.DiscardUnknown(m)
}

var xxx_messageInfo_Query proto.InternalMessageInfo

func (m *Query) GetIndex() *Index {
	if m != nil {
		return m.Index
	}
	return nil
}

func (m *Query) GetFilters() []*Filter {
	if m != nil {
		return m.Filters
	}
	return nil
}

func init() {
	proto.RegisterType((*Ticket)(nil), "api.Ticket")
	proto.RegisterMapType((map[string]float64)(nil), "api.Ticket.DargsEntry")
	proto.RegisterType((*Assignment)(nil), "api.Assignment")
	proto.RegisterType((*Index)(nil), "api.Index")
	proto.RegisterType((*PoolIndex)(nil), "api.PoolIndex")
	proto.RegisterType((*RangeIndex)(nil), "api.RangeIndex")
	proto.RegisterType((*Filter)(nil), "api.Filter")
	proto.RegisterType((*RangeFilter)(nil), "api.RangeFilter")
	proto.RegisterType((*Query)(nil), "api.Query")
}

func init() { proto.RegisterFile("api/messages.proto", fileDescriptor_cb9fb1f207fd5b8c) }

var fileDescriptor_cb9fb1f207fd5b8c = []byte{
	// 434 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0xac, 0x52, 0x4d, 0x8b, 0xdb, 0x30,
	0x10, 0x5d, 0xc7, 0x75, 0x96, 0x8c, 0xcb, 0x76, 0x19, 0x42, 0x49, 0xf7, 0x50, 0x8c, 0xa1, 0x90,
	0x43, 0xb1, 0x21, 0x6d, 0x61, 0xe9, 0x6d, 0x97, 0x6e, 0x49, 0x6f, 0x5b, 0xb1, 0xa7, 0x5e, 0x8a,
	0x92, 0x4c, 0x8c, 0x58, 0x47, 0x12, 0xb2, 0x5c, 0xe2, 0x3f, 0xd6, 0xdf, 0x57, 0x24, 0x39, 0xb1,
	0x0f, 0x3d, 0x95, 0xde, 0x34, 0x6f, 0xde, 0xbc, 0x37, 0x1f, 0x02, 0xe4, 0x5a, 0x94, 0x07, 0x6a,
	0x1a, 0x5e, 0x51, 0x53, 0x68, 0xa3, 0xac, 0xc2, 0x98, 0x6b, 0x71, 0xf3, 0xa6, 0x52, 0xaa, 0xaa,
	0xa9, 0xf4, 0xd0, 0xa6, 0xdd, 0x97, 0x5c, 0x76, 0x21, 0x9f, 0xff, 0x8e, 0x60, 0xfa, 0x24, 0xb6,
	0xcf, 0x64, 0xf1, 0x0a, 0x26, 0x62, 0xb7, 0x88, 0xb2, 0x68, 0x39, 0x63, 0x13, 0xb1, 0xc3, 0x8f,
	0x00, 0xda, 0x28, 0x4d, 0xc6, 0x0a, 0x6a, 0x16, 0x93, 0x2c, 0x5a, 0xa6, 0xab, 0x79, 0x11, 0xa4,
	0x8a, 0x93, 0x54, 0x71, 0x27, 0x3b, 0x36, 0xe2, 0xe1, 0x7b, 0x48, 0x76, 0xdc, 0x54, 0xcd, 0x22,
	0xce, 0xe2, 0x65, 0xba, 0x7a, 0x5d, 0x70, 0x2d, 0x8a, 0xe0, 0x50, 0x7c, 0x71, 0x89, 0x07, 0x69,
	0x4d, 0xc7, 0x02, 0xe9, 0xe6, 0x16, 0x60, 0x00, 0xf1, 0x1a, 0xe2, 0x67, 0xea, 0xfa, 0x16, 0xdc,
	0x13, 0xe7, 0x90, 0xfc, 0xe2, 0x75, 0x4b, 0xde, 0x3e, 0x62, 0x21, 0xf8, 0x3c, 0xb9, 0x8d, 0xf2,
	0x23, 0xc0, 0x5d, 0xd3, 0x88, 0x4a, 0x1e, 0x48, 0x5a, 0x7c, 0x0b, 0xb0, 0x55, 0x52, 0xd2, 0xd6,
	0x0a, 0x25, 0x7b, 0x81, 0x11, 0xf2, 0x8f, 0xb3, 0xcc, 0x21, 0x21, 0x63, 0x94, 0x59, 0xc4, 0x5e,
	0x30, 0x04, 0x79, 0x0b, 0xc9, 0x37, 0xb9, 0xa3, 0x23, 0x96, 0x00, 0x5a, 0xa9, 0xfa, 0xa7, 0x70,
	0x91, 0x37, 0x4d, 0x57, 0x57, 0x7e, 0xde, 0x47, 0xa5, 0x6a, 0xcf, 0x59, 0x5f, 0xb0, 0x99, 0x3e,
	0x05, 0xb8, 0x82, 0xd4, 0x70, 0x59, 0x51, 0x5f, 0x11, 0xda, 0x78, 0xe5, 0x2b, 0x98, 0xc3, 0x4f,
	0x25, 0x60, 0xce, 0xd1, 0xfd, 0x65, 0xbf, 0x81, 0xfc, 0x09, 0x66, 0x67, 0x59, 0xcc, 0x20, 0x19,
	0xbb, 0x82, 0xd7, 0xf0, 0x29, 0x16, 0x12, 0xf8, 0x0e, 0x2e, 0xf7, 0xa2, 0xb6, 0x64, 0xdc, 0xb8,
	0xee, 0x12, 0xa9, 0xe7, 0x7c, 0xf5, 0x18, 0x3b, 0xe5, 0xf2, 0x0c, 0x60, 0xb0, 0x46, 0x84, 0x17,
	0xee, 0x2e, 0xfd, 0x02, 0xfd, 0x3b, 0x5f, 0xc3, 0x34, 0x14, 0xe1, 0x27, 0x78, 0x19, 0xda, 0x0f,
	0xc5, 0xbd, 0xf7, 0xf5, 0xd0, 0x7f, 0xe0, 0xad, 0x2f, 0x58, 0x18, 0x33, 0x84, 0xc3, 0x04, 0x0f,
	0x90, 0x8e, 0x68, 0x7f, 0x33, 0x73, 0x3f, 0xe0, 0x20, 0x64, 0x7f, 0x6d, 0xf7, 0xf4, 0x08, 0x3f,
	0xfa, 0x0b, 0x38, 0x84, 0x1f, 0xf3, 0x47, 0x48, 0xbe, 0xb7, 0x64, 0xba, 0xff, 0xb6, 0x84, 0xfb,
	0xf9, 0x0f, 0x14, 0xd2, 0x92, 0x91, 0xbc, 0x2e, 0xf7, 0xad, 0x6d, 0x0d, 0x95, 0x7a, 0xb3, 0x99,
	0xfa, 0x7f, 0xf1, 0xe1, 0x4f, 0x00, 0x00, 0x00, 0xff, 0xff, 0xe3, 0xad, 0xf3, 0x56, 0x57, 0x03,
	0x00, 0x00,
}
