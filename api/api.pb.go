// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.8
// source: api/api.proto

package api

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	durationpb "google.golang.org/protobuf/types/known/durationpb"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Cmd struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Cmd  string   `protobuf:"bytes,1,opt,name=cmd,proto3" json:"cmd,omitempty"`
	Args []string `protobuf:"bytes,2,rep,name=args,proto3" json:"args,omitempty"`
}

func (x *Cmd) Reset() {
	*x = Cmd{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Cmd) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Cmd) ProtoMessage() {}

func (x *Cmd) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Cmd.ProtoReflect.Descriptor instead.
func (*Cmd) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{0}
}

func (x *Cmd) GetCmd() string {
	if x != nil {
		return x.Cmd
	}
	return ""
}

func (x *Cmd) GetArgs() []string {
	if x != nil {
		return x.Args
	}
	return nil
}

type ConfigStart struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config string   `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
	Names  []string `protobuf:"bytes,2,rep,name=names,proto3" json:"names,omitempty"`
}

func (x *ConfigStart) Reset() {
	*x = ConfigStart{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ConfigStart) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ConfigStart) ProtoMessage() {}

func (x *ConfigStart) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ConfigStart.ProtoReflect.Descriptor instead.
func (*ConfigStart) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{1}
}

func (x *ConfigStart) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}

func (x *ConfigStart) GetNames() []string {
	if x != nil {
		return x.Names
	}
	return nil
}

type StartReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name  string   `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	Cwd   string   `protobuf:"bytes,2,opt,name=cwd,proto3" json:"cwd,omitempty"`
	Tags  *Tags    `protobuf:"bytes,3,opt,name=tags,proto3" json:"tags,omitempty"`
	Watch []string `protobuf:"bytes,4,rep,name=watch,proto3" json:"watch,omitempty"`
	Cmd   string   `protobuf:"bytes,5,opt,name=cmd,proto3" json:"cmd,omitempty"`
}

func (x *StartReq) Reset() {
	*x = StartReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartReq) ProtoMessage() {}

func (x *StartReq) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartReq.ProtoReflect.Descriptor instead.
func (*StartReq) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{2}
}

func (x *StartReq) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *StartReq) GetCwd() string {
	if x != nil {
		return x.Cwd
	}
	return ""
}

func (x *StartReq) GetTags() *Tags {
	if x != nil {
		return x.Tags
	}
	return nil
}

func (x *StartReq) GetWatch() []string {
	if x != nil {
		return x.Watch
	}
	return nil
}

func (x *StartReq) GetCmd() string {
	if x != nil {
		return x.Cmd
	}
	return ""
}

type StartResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id  uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Pid int64  `protobuf:"varint,2,opt,name=pid,proto3" json:"pid,omitempty"`
}

func (x *StartResp) Reset() {
	*x = StartResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StartResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StartResp) ProtoMessage() {}

func (x *StartResp) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use StartResp.ProtoReflect.Descriptor instead.
func (*StartResp) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{3}
}

func (x *StartResp) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *StartResp) GetPid() int64 {
	if x != nil {
		return x.Pid
	}
	return 0
}

type RunningInfo struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pid    int64                `protobuf:"varint,1,opt,name=pid,proto3" json:"pid,omitempty"`
	Uptime *durationpb.Duration `protobuf:"bytes,4,opt,name=uptime,proto3" json:"uptime,omitempty"`
}

func (x *RunningInfo) Reset() {
	*x = RunningInfo{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunningInfo) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunningInfo) ProtoMessage() {}

func (x *RunningInfo) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunningInfo.ProtoReflect.Descriptor instead.
func (*RunningInfo) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{4}
}

func (x *RunningInfo) GetPid() int64 {
	if x != nil {
		return x.Pid
	}
	return 0
}

func (x *RunningInfo) GetUptime() *durationpb.Duration {
	if x != nil {
		return x.Uptime
	}
	return nil
}

type ListResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Items []*ListRespEntry `protobuf:"bytes,1,rep,name=items,proto3" json:"items,omitempty"`
}

func (x *ListResp) Reset() {
	*x = ListResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListResp) ProtoMessage() {}

func (x *ListResp) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListResp.ProtoReflect.Descriptor instead.
func (*ListResp) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{5}
}

func (x *ListResp) GetItems() []*ListRespEntry {
	if x != nil {
		return x.Items
	}
	return nil
}

type ListRespEntry struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id   uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Cmd  string `protobuf:"bytes,2,opt,name=cmd,proto3" json:"cmd,omitempty"`
	Name string `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	// Types that are assignable to Status:
	//
	//	*ListRespEntry_Stopped
	//	*ListRespEntry_Running
	//	*ListRespEntry_Errored
	//	*ListRespEntry_Invalid
	Status isListRespEntry_Status `protobuf_oneof:"status"`
	Tags   *Tags                  `protobuf:"bytes,5,opt,name=tags,proto3" json:"tags,omitempty"`
	Cpu    int64                  `protobuf:"varint,6,opt,name=cpu,proto3" json:"cpu,omitempty"`       // round(cpu usage in % * 100)
	Memory int64                  `protobuf:"varint,7,opt,name=memory,proto3" json:"memory,omitempty"` // in bytes
}

func (x *ListRespEntry) Reset() {
	*x = ListRespEntry{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ListRespEntry) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ListRespEntry) ProtoMessage() {}

func (x *ListRespEntry) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ListRespEntry.ProtoReflect.Descriptor instead.
func (*ListRespEntry) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{6}
}

func (x *ListRespEntry) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *ListRespEntry) GetCmd() string {
	if x != nil {
		return x.Cmd
	}
	return ""
}

func (x *ListRespEntry) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (m *ListRespEntry) GetStatus() isListRespEntry_Status {
	if m != nil {
		return m.Status
	}
	return nil
}

func (x *ListRespEntry) GetStopped() *emptypb.Empty {
	if x, ok := x.GetStatus().(*ListRespEntry_Stopped); ok {
		return x.Stopped
	}
	return nil
}

func (x *ListRespEntry) GetRunning() *RunningInfo {
	if x, ok := x.GetStatus().(*ListRespEntry_Running); ok {
		return x.Running
	}
	return nil
}

func (x *ListRespEntry) GetErrored() *emptypb.Empty {
	if x, ok := x.GetStatus().(*ListRespEntry_Errored); ok {
		return x.Errored
	}
	return nil
}

func (x *ListRespEntry) GetInvalid() string {
	if x, ok := x.GetStatus().(*ListRespEntry_Invalid); ok {
		return x.Invalid
	}
	return ""
}

func (x *ListRespEntry) GetTags() *Tags {
	if x != nil {
		return x.Tags
	}
	return nil
}

func (x *ListRespEntry) GetCpu() int64 {
	if x != nil {
		return x.Cpu
	}
	return 0
}

func (x *ListRespEntry) GetMemory() int64 {
	if x != nil {
		return x.Memory
	}
	return 0
}

type isListRespEntry_Status interface {
	isListRespEntry_Status()
}

type ListRespEntry_Stopped struct {
	Stopped *emptypb.Empty `protobuf:"bytes,10,opt,name=stopped,proto3,oneof"`
}

type ListRespEntry_Running struct {
	Running *RunningInfo `protobuf:"bytes,11,opt,name=running,proto3,oneof"`
}

type ListRespEntry_Errored struct {
	Errored *emptypb.Empty `protobuf:"bytes,12,opt,name=errored,proto3,oneof"`
}

type ListRespEntry_Invalid struct {
	Invalid string `protobuf:"bytes,13,opt,name=invalid,proto3,oneof"`
}

func (*ListRespEntry_Stopped) isListRespEntry_Status() {}

func (*ListRespEntry_Running) isListRespEntry_Status() {}

func (*ListRespEntry_Errored) isListRespEntry_Status() {}

func (*ListRespEntry_Invalid) isListRespEntry_Status() {}

type DeleteReq struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Filters []*DeleteFilter `protobuf:"bytes,1,rep,name=filters,proto3" json:"filters,omitempty"`
}

func (x *DeleteReq) Reset() {
	*x = DeleteReq{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteReq) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteReq) ProtoMessage() {}

func (x *DeleteReq) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteReq.ProtoReflect.Descriptor instead.
func (*DeleteReq) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{7}
}

func (x *DeleteReq) GetFilters() []*DeleteFilter {
	if x != nil {
		return x.Filters
	}
	return nil
}

type DeleteFilter struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Filter:
	//
	//	*DeleteFilter_Tags
	//	*DeleteFilter_Name
	//	*DeleteFilter_All
	//	*DeleteFilter_Config
	Filter isDeleteFilter_Filter `protobuf_oneof:"filter"`
}

func (x *DeleteFilter) Reset() {
	*x = DeleteFilter{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteFilter) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteFilter) ProtoMessage() {}

func (x *DeleteFilter) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[8]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteFilter.ProtoReflect.Descriptor instead.
func (*DeleteFilter) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{8}
}

func (m *DeleteFilter) GetFilter() isDeleteFilter_Filter {
	if m != nil {
		return m.Filter
	}
	return nil
}

func (x *DeleteFilter) GetTags() *Tags {
	if x, ok := x.GetFilter().(*DeleteFilter_Tags); ok {
		return x.Tags
	}
	return nil
}

func (x *DeleteFilter) GetName() string {
	if x, ok := x.GetFilter().(*DeleteFilter_Name); ok {
		return x.Name
	}
	return ""
}

func (x *DeleteFilter) GetAll() *emptypb.Empty {
	if x, ok := x.GetFilter().(*DeleteFilter_All); ok {
		return x.All
	}
	return nil
}

func (x *DeleteFilter) GetConfig() string {
	if x, ok := x.GetFilter().(*DeleteFilter_Config); ok {
		return x.Config
	}
	return ""
}

type isDeleteFilter_Filter interface {
	isDeleteFilter_Filter()
}

type DeleteFilter_Tags struct {
	Tags *Tags `protobuf:"bytes,10,opt,name=tags,proto3,oneof"` // all procs having all of those tags
}

type DeleteFilter_Name struct {
	Name string `protobuf:"bytes,11,opt,name=name,proto3,oneof"` // proc with such name
}

type DeleteFilter_All struct {
	All *emptypb.Empty `protobuf:"bytes,12,opt,name=all,proto3,oneof"` // all
}

type DeleteFilter_Config struct {
	Config string `protobuf:"bytes,13,opt,name=config,proto3,oneof"` // all procs described in config
}

func (*DeleteFilter_Tags) isDeleteFilter_Filter() {}

func (*DeleteFilter_Name) isDeleteFilter_Filter() {}

func (*DeleteFilter_All) isDeleteFilter_Filter() {}

func (*DeleteFilter_Config) isDeleteFilter_Filter() {}

type Tags struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Tags []string `protobuf:"bytes,1,rep,name=tags,proto3" json:"tags,omitempty"`
}

func (x *Tags) Reset() {
	*x = Tags{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Tags) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Tags) ProtoMessage() {}

func (x *Tags) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[9]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Tags.ProtoReflect.Descriptor instead.
func (*Tags) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{9}
}

func (x *Tags) GetTags() []string {
	if x != nil {
		return x.Tags
	}
	return nil
}

type DeleteResp struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id []uint64 `protobuf:"varint,1,rep,packed,name=id,proto3" json:"id,omitempty"`
}

func (x *DeleteResp) Reset() {
	*x = DeleteResp{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[10]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *DeleteResp) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*DeleteResp) ProtoMessage() {}

func (x *DeleteResp) ProtoReflect() protoreflect.Message {
	mi := &file_api_api_proto_msgTypes[10]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use DeleteResp.ProtoReflect.Descriptor instead.
func (*DeleteResp) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{10}
}

func (x *DeleteResp) GetId() []uint64 {
	if x != nil {
		return x.Id
	}
	return nil
}

var File_api_api_proto protoreflect.FileDescriptor

var file_api_api_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x03, 0x61, 0x70, 0x69, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x64, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x22, 0x2b, 0x0a, 0x03, 0x43, 0x6d, 0x64, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x6d, 0x64, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x63, 0x6d, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x61, 0x72,
	0x67, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x61, 0x72, 0x67, 0x73, 0x22, 0x3b,
	0x0a, 0x0b, 0x43, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x53, 0x74, 0x61, 0x72, 0x74, 0x12, 0x16, 0x0a,
	0x06, 0x63, 0x6f, 0x6e, 0x66, 0x69, 0x67, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x06, 0x63,
	0x6f, 0x6e, 0x66, 0x69, 0x67, 0x12, 0x14, 0x0a, 0x05, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x18, 0x02,
	0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x6e, 0x61, 0x6d, 0x65, 0x73, 0x22, 0x77, 0x0a, 0x08, 0x53,
	0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x71, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18,
	0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x10, 0x0a, 0x03, 0x63,
	0x77, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x63, 0x77, 0x64, 0x12, 0x1d, 0x0a,
	0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x54, 0x61, 0x67, 0x73, 0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x12, 0x14, 0x0a, 0x05,
	0x77, 0x61, 0x74, 0x63, 0x68, 0x18, 0x04, 0x20, 0x03, 0x28, 0x09, 0x52, 0x05, 0x77, 0x61, 0x74,
	0x63, 0x68, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x6d, 0x64, 0x18, 0x05, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x03, 0x63, 0x6d, 0x64, 0x22, 0x2d, 0x0a, 0x09, 0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x10, 0x0a, 0x03, 0x70, 0x69, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x03, 0x52, 0x03,
	0x70, 0x69, 0x64, 0x22, 0x52, 0x0a, 0x0b, 0x52, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x49, 0x6e,
	0x66, 0x6f, 0x12, 0x10, 0x0a, 0x03, 0x70, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x03, 0x70, 0x69, 0x64, 0x12, 0x31, 0x0a, 0x06, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65, 0x18, 0x04,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x44, 0x75, 0x72, 0x61, 0x74, 0x69, 0x6f, 0x6e, 0x52,
	0x06, 0x75, 0x70, 0x74, 0x69, 0x6d, 0x65, 0x22, 0x34, 0x0a, 0x08, 0x4c, 0x69, 0x73, 0x74, 0x52,
	0x65, 0x73, 0x70, 0x12, 0x28, 0x0a, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x0b, 0x32, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73,
	0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x05, 0x69, 0x74, 0x65, 0x6d, 0x73, 0x22, 0xca, 0x02,
	0x0a, 0x0d, 0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x12,
	0x10, 0x0a, 0x03, 0x63, 0x6d, 0x64, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x63, 0x6d,
	0x64, 0x12, 0x12, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x32, 0x0a, 0x07, 0x73, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64,
	0x18, 0x0a, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x48, 0x00,
	0x52, 0x07, 0x73, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x12, 0x2c, 0x0a, 0x07, 0x72, 0x75, 0x6e,
	0x6e, 0x69, 0x6e, 0x67, 0x18, 0x0b, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x52, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x49, 0x6e, 0x66, 0x6f, 0x48, 0x00, 0x52, 0x07,
	0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x32, 0x0a, 0x07, 0x65, 0x72, 0x72, 0x6f, 0x72,
	0x65, 0x64, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x48, 0x00, 0x52, 0x07, 0x65, 0x72, 0x72, 0x6f, 0x72, 0x65, 0x64, 0x12, 0x1a, 0x0a, 0x07, 0x69,
	0x6e, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x07,
	0x69, 0x6e, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x12, 0x1d, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18,
	0x05, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x54, 0x61, 0x67, 0x73,
	0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x70, 0x75, 0x18, 0x06, 0x20,
	0x01, 0x28, 0x03, 0x52, 0x03, 0x63, 0x70, 0x75, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x6d, 0x6f,
	0x72, 0x79, 0x18, 0x07, 0x20, 0x01, 0x28, 0x03, 0x52, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79,
	0x42, 0x08, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x38, 0x0a, 0x09, 0x44, 0x65,
	0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x12, 0x2b, 0x0a, 0x07, 0x66, 0x69, 0x6c, 0x74, 0x65,
	0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x11, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x44,
	0x65, 0x6c, 0x65, 0x74, 0x65, 0x46, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x52, 0x07, 0x66, 0x69, 0x6c,
	0x74, 0x65, 0x72, 0x73, 0x22, 0x95, 0x01, 0x0a, 0x0c, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x46,
	0x69, 0x6c, 0x74, 0x65, 0x72, 0x12, 0x1f, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x0a, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x09, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x54, 0x61, 0x67, 0x73, 0x48, 0x00,
	0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x12, 0x14, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x0b,
	0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x12, 0x2a, 0x0a, 0x03,
	0x61, 0x6c, 0x6c, 0x18, 0x0c, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67,
	0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74,
	0x79, 0x48, 0x00, 0x52, 0x03, 0x61, 0x6c, 0x6c, 0x12, 0x18, 0x0a, 0x06, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x18, 0x0d, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00, 0x52, 0x06, 0x63, 0x6f, 0x6e, 0x66,
	0x69, 0x67, 0x42, 0x08, 0x0a, 0x06, 0x66, 0x69, 0x6c, 0x74, 0x65, 0x72, 0x22, 0x1a, 0x0a, 0x04,
	0x54, 0x61, 0x67, 0x73, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x09, 0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x22, 0x1c, 0x0a, 0x0a, 0x44, 0x65, 0x6c, 0x65,
	0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x12, 0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x03,
	0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x32, 0xb3, 0x01, 0x0a, 0x06, 0x44, 0x61, 0x65, 0x6d, 0x6f,
	0x6e, 0x12, 0x26, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x72, 0x74, 0x12, 0x0d, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x71, 0x1a, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x53, 0x74, 0x61, 0x72, 0x74, 0x52, 0x65, 0x73, 0x70, 0x12, 0x2d, 0x0a, 0x04, 0x4c, 0x69, 0x73,
	0x74, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f,
	0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x0d, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x4c, 0x69, 0x73, 0x74, 0x52, 0x65, 0x73, 0x70, 0x12, 0x27, 0x0a, 0x04, 0x53, 0x74, 0x6f, 0x70,
	0x12, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71,
	0x1a, 0x0f, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x73,
	0x70, 0x12, 0x29, 0x0a, 0x06, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x0e, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x71, 0x1a, 0x0f, 0x2e, 0x61, 0x70,
	0x69, 0x2e, 0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x42, 0x1c, 0x5a, 0x1a,
	0x67, 0x69, 0x74, 0x68, 0x75, 0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x70, 0x72, 0x74, 0x72,
	0x32, 0x35, 0x38, 0x2f, 0x70, 0x6d, 0x2f, 0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x33,
}

var (
	file_api_api_proto_rawDescOnce sync.Once
	file_api_api_proto_rawDescData = file_api_api_proto_rawDesc
)

func file_api_api_proto_rawDescGZIP() []byte {
	file_api_api_proto_rawDescOnce.Do(func() {
		file_api_api_proto_rawDescData = protoimpl.X.CompressGZIP(file_api_api_proto_rawDescData)
	})
	return file_api_api_proto_rawDescData
}

var file_api_api_proto_msgTypes = make([]protoimpl.MessageInfo, 11)
var file_api_api_proto_goTypes = []interface{}{
	(*Cmd)(nil),                 // 0: api.Cmd
	(*ConfigStart)(nil),         // 1: api.ConfigStart
	(*StartReq)(nil),            // 2: api.StartReq
	(*StartResp)(nil),           // 3: api.StartResp
	(*RunningInfo)(nil),         // 4: api.RunningInfo
	(*ListResp)(nil),            // 5: api.ListResp
	(*ListRespEntry)(nil),       // 6: api.ListRespEntry
	(*DeleteReq)(nil),           // 7: api.DeleteReq
	(*DeleteFilter)(nil),        // 8: api.DeleteFilter
	(*Tags)(nil),                // 9: api.Tags
	(*DeleteResp)(nil),          // 10: api.DeleteResp
	(*durationpb.Duration)(nil), // 11: google.protobuf.Duration
	(*emptypb.Empty)(nil),       // 12: google.protobuf.Empty
}
var file_api_api_proto_depIdxs = []int32{
	9,  // 0: api.StartReq.tags:type_name -> api.Tags
	11, // 1: api.RunningInfo.uptime:type_name -> google.protobuf.Duration
	6,  // 2: api.ListResp.items:type_name -> api.ListRespEntry
	12, // 3: api.ListRespEntry.stopped:type_name -> google.protobuf.Empty
	4,  // 4: api.ListRespEntry.running:type_name -> api.RunningInfo
	12, // 5: api.ListRespEntry.errored:type_name -> google.protobuf.Empty
	9,  // 6: api.ListRespEntry.tags:type_name -> api.Tags
	8,  // 7: api.DeleteReq.filters:type_name -> api.DeleteFilter
	9,  // 8: api.DeleteFilter.tags:type_name -> api.Tags
	12, // 9: api.DeleteFilter.all:type_name -> google.protobuf.Empty
	2,  // 10: api.Daemon.Start:input_type -> api.StartReq
	12, // 11: api.Daemon.List:input_type -> google.protobuf.Empty
	7,  // 12: api.Daemon.Stop:input_type -> api.DeleteReq
	7,  // 13: api.Daemon.Delete:input_type -> api.DeleteReq
	3,  // 14: api.Daemon.Start:output_type -> api.StartResp
	5,  // 15: api.Daemon.List:output_type -> api.ListResp
	10, // 16: api.Daemon.Stop:output_type -> api.DeleteResp
	10, // 17: api.Daemon.Delete:output_type -> api.DeleteResp
	14, // [14:18] is the sub-list for method output_type
	10, // [10:14] is the sub-list for method input_type
	10, // [10:10] is the sub-list for extension type_name
	10, // [10:10] is the sub-list for extension extendee
	0,  // [0:10] is the sub-list for field type_name
}

func init() { file_api_api_proto_init() }
func file_api_api_proto_init() {
	if File_api_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Cmd); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ConfigStart); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*StartResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunningInfo); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ListRespEntry); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteReq); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[8].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteFilter); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[9].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Tags); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_api_api_proto_msgTypes[10].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*DeleteResp); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_api_api_proto_msgTypes[6].OneofWrappers = []interface{}{
		(*ListRespEntry_Stopped)(nil),
		(*ListRespEntry_Running)(nil),
		(*ListRespEntry_Errored)(nil),
		(*ListRespEntry_Invalid)(nil),
	}
	file_api_api_proto_msgTypes[8].OneofWrappers = []interface{}{
		(*DeleteFilter_Tags)(nil),
		(*DeleteFilter_Name)(nil),
		(*DeleteFilter_All)(nil),
		(*DeleteFilter_Config)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_api_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   11,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_api_proto_goTypes,
		DependencyIndexes: file_api_api_proto_depIdxs,
		MessageInfos:      file_api_api_proto_msgTypes,
	}.Build()
	File_api_api_proto = out.File
	file_api_api_proto_rawDesc = nil
	file_api_api_proto_goTypes = nil
	file_api_api_proto_depIdxs = nil
}
