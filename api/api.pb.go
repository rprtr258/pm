// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.15.8
// source: api/api.proto

package api

import (
	reflect "reflect"
	sync "sync"

	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Signal int32

const (
	Signal_SIGNAL_UNSPECIFIED Signal = 0
	Signal_SIGNAL_SIGTERM     Signal = 1
	Signal_SIGNAL_SIGKILL     Signal = 2
)

// Enum value maps for Signal.
var (
	Signal_name = map[int32]string{
		0: "SIGNAL_UNSPECIFIED",
		1: "SIGNAL_SIGTERM",
		2: "SIGNAL_SIGKILL",
	}
	Signal_value = map[string]int32{
		"SIGNAL_UNSPECIFIED": 0,
		"SIGNAL_SIGTERM":     1,
		"SIGNAL_SIGKILL":     2,
	}
)

func (x Signal) Enum() *Signal {
	p := new(Signal)
	*p = x
	return p
}

func (x Signal) String() string {
	return protoimpl.X.EnumStringOf(x.Descriptor(), protoreflect.EnumNumber(x))
}

func (Signal) Descriptor() protoreflect.EnumDescriptor {
	return file_api_api_proto_enumTypes[0].Descriptor()
}

func (Signal) Type() protoreflect.EnumType {
	return &file_api_api_proto_enumTypes[0]
}

func (x Signal) Number() protoreflect.EnumNumber {
	return protoreflect.EnumNumber(x)
}

// Deprecated: Use Signal.Descriptor instead.
func (Signal) EnumDescriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{0}
}

type ProcessID struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id uint64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (x *ProcessID) Reset() {
	*x = ProcessID{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProcessID) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessID) ProtoMessage() {}

func (x *ProcessID) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use ProcessID.ProtoReflect.Descriptor instead.
func (*ProcessID) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{0}
}

func (x *ProcessID) GetId() uint64 {
	if x != nil {
		return x.Id
	}
	return 0
}

type CreateRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Options []*ProcessOptions `protobuf:"bytes,1,rep,name=options,proto3" json:"options,omitempty"`
}

func (x *CreateRequest) Reset() {
	*x = CreateRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *CreateRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*CreateRequest) ProtoMessage() {}

func (x *CreateRequest) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use CreateRequest.ProtoReflect.Descriptor instead.
func (*CreateRequest) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{1}
}

func (x *CreateRequest) GetOptions() []*ProcessOptions {
	if x != nil {
		return x.Options
	}
	return nil
}

type ProcessOptions struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Command string   `protobuf:"bytes,1,opt,name=command,proto3" json:"command,omitempty"`
	Args    []string `protobuf:"bytes,2,rep,name=args,proto3" json:"args,omitempty"`
	Name    *string  `protobuf:"bytes,3,opt,name=name,proto3,oneof" json:"name,omitempty"`
	Cwd     string   `protobuf:"bytes,4,opt,name=cwd,proto3" json:"cwd,omitempty"`
	Tags    []string `protobuf:"bytes,5,rep,name=tags,proto3" json:"tags,omitempty"`
}

func (x *ProcessOptions) Reset() {
	*x = ProcessOptions{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProcessOptions) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessOptions) ProtoMessage() {}

func (x *ProcessOptions) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use ProcessOptions.ProtoReflect.Descriptor instead.
func (*ProcessOptions) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{2}
}

func (x *ProcessOptions) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

func (x *ProcessOptions) GetArgs() []string {
	if x != nil {
		return x.Args
	}
	return nil
}

func (x *ProcessOptions) GetName() string {
	if x != nil && x.Name != nil {
		return *x.Name
	}
	return ""
}

func (x *ProcessOptions) GetCwd() string {
	if x != nil {
		return x.Cwd
	}
	return ""
}

func (x *ProcessOptions) GetTags() []string {
	if x != nil {
		return x.Tags
	}
	return nil
}

type ProcessesList struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Processes []*Process `protobuf:"bytes,1,rep,name=processes,proto3" json:"processes,omitempty"`
}

func (x *ProcessesList) Reset() {
	*x = ProcessesList{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProcessesList) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessesList) ProtoMessage() {}

func (x *ProcessesList) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use ProcessesList.ProtoReflect.Descriptor instead.
func (*ProcessesList) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{3}
}

func (x *ProcessesList) GetProcesses() []*Process {
	if x != nil {
		return x.Processes
	}
	return nil
}

type Process struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      *ProcessID     `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Status  *ProcessStatus `protobuf:"bytes,2,opt,name=status,proto3" json:"status,omitempty"`
	Name    string         `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Cwd     string         `protobuf:"bytes,4,opt,name=cwd,proto3" json:"cwd,omitempty"`
	Tags    []string       `protobuf:"bytes,5,rep,name=tags,proto3" json:"tags,omitempty"`
	Command string         `protobuf:"bytes,6,opt,name=command,proto3" json:"command,omitempty"`
	Args    []string       `protobuf:"bytes,7,rep,name=args,proto3" json:"args,omitempty"`
}

func (x *Process) Reset() {
	*x = Process{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Process) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Process) ProtoMessage() {}

func (x *Process) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use Process.ProtoReflect.Descriptor instead.
func (*Process) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{4}
}

func (x *Process) GetId() *ProcessID {
	if x != nil {
		return x.Id
	}
	return nil
}

func (x *Process) GetStatus() *ProcessStatus {
	if x != nil {
		return x.Status
	}
	return nil
}

func (x *Process) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *Process) GetCwd() string {
	if x != nil {
		return x.Cwd
	}
	return ""
}

func (x *Process) GetTags() []string {
	if x != nil {
		return x.Tags
	}
	return nil
}

func (x *Process) GetCommand() string {
	if x != nil {
		return x.Command
	}
	return ""
}

func (x *Process) GetArgs() []string {
	if x != nil {
		return x.Args
	}
	return nil
}

type ProcessStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Status:
	//
	//	*ProcessStatus_Invalid
	//	*ProcessStatus_Starting
	//	*ProcessStatus_Running
	//	*ProcessStatus_Stopped
	Status isProcessStatus_Status `protobuf_oneof:"status"`
}

func (x *ProcessStatus) Reset() {
	*x = ProcessStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ProcessStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ProcessStatus) ProtoMessage() {}

func (x *ProcessStatus) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use ProcessStatus.ProtoReflect.Descriptor instead.
func (*ProcessStatus) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{5}
}

func (m *ProcessStatus) GetStatus() isProcessStatus_Status {
	if m != nil {
		return m.Status
	}
	return nil
}

func (x *ProcessStatus) GetInvalid() *emptypb.Empty {
	if x, ok := x.GetStatus().(*ProcessStatus_Invalid); ok {
		return x.Invalid
	}
	return nil
}

func (x *ProcessStatus) GetStarting() *emptypb.Empty {
	if x, ok := x.GetStatus().(*ProcessStatus_Starting); ok {
		return x.Starting
	}
	return nil
}

func (x *ProcessStatus) GetRunning() *RunningProcessStatus {
	if x, ok := x.GetStatus().(*ProcessStatus_Running); ok {
		return x.Running
	}
	return nil
}

func (x *ProcessStatus) GetStopped() *StoppedProcessStatus {
	if x, ok := x.GetStatus().(*ProcessStatus_Stopped); ok {
		return x.Stopped
	}
	return nil
}

type isProcessStatus_Status interface {
	isProcessStatus_Status()
}

type ProcessStatus_Invalid struct {
	Invalid *emptypb.Empty `protobuf:"bytes,1,opt,name=invalid,proto3,oneof"`
}

type ProcessStatus_Starting struct {
	Starting *emptypb.Empty `protobuf:"bytes,2,opt,name=starting,proto3,oneof"`
}

type ProcessStatus_Running struct {
	Running *RunningProcessStatus `protobuf:"bytes,3,opt,name=running,proto3,oneof"`
}

type ProcessStatus_Stopped struct {
	Stopped *StoppedProcessStatus `protobuf:"bytes,4,opt,name=stopped,proto3,oneof"`
}

func (*ProcessStatus_Invalid) isProcessStatus_Status() {}

func (*ProcessStatus_Starting) isProcessStatus_Status() {}

func (*ProcessStatus_Running) isProcessStatus_Status() {}

func (*ProcessStatus_Stopped) isProcessStatus_Status() {}

type RunningProcessStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Pid       int64                  `protobuf:"varint,1,opt,name=pid,proto3" json:"pid,omitempty"`
	StartTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	Cpu       uint64                 `protobuf:"varint,3,opt,name=cpu,proto3" json:"cpu,omitempty"`
	Memory    uint64                 `protobuf:"varint,4,opt,name=memory,proto3" json:"memory,omitempty"`
}

func (x *RunningProcessStatus) Reset() {
	*x = RunningProcessStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunningProcessStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunningProcessStatus) ProtoMessage() {}

func (x *RunningProcessStatus) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use RunningProcessStatus.ProtoReflect.Descriptor instead.
func (*RunningProcessStatus) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{6}
}

func (x *RunningProcessStatus) GetPid() int64 {
	if x != nil {
		return x.Pid
	}
	return 0
}

func (x *RunningProcessStatus) GetStartTime() *timestamppb.Timestamp {
	if x != nil {
		return x.StartTime
	}
	return nil
}

func (x *RunningProcessStatus) GetCpu() uint64 {
	if x != nil {
		return x.Cpu
	}
	return 0
}

func (x *RunningProcessStatus) GetMemory() uint64 {
	if x != nil {
		return x.Memory
	}
	return 0
}

type StoppedProcessStatus struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	ExitCode  int64                  `protobuf:"varint,1,opt,name=exit_code,json=exitCode,proto3" json:"exit_code,omitempty"`
	StoppedAt *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=stopped_at,json=stoppedAt,proto3" json:"stopped_at,omitempty"`
}

func (x *StoppedProcessStatus) Reset() {
	*x = StoppedProcessStatus{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *StoppedProcessStatus) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*StoppedProcessStatus) ProtoMessage() {}

func (x *StoppedProcessStatus) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use StoppedProcessStatus.ProtoReflect.Descriptor instead.
func (*StoppedProcessStatus) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{7}
}

func (x *StoppedProcessStatus) GetExitCode() int64 {
	if x != nil {
		return x.ExitCode
	}
	return 0
}

func (x *StoppedProcessStatus) GetStoppedAt() *timestamppb.Timestamp {
	if x != nil {
		return x.StoppedAt
	}
	return nil
}

type IDs struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ids []*ProcessID `protobuf:"bytes,1,rep,name=ids,proto3" json:"ids,omitempty"`
}

func (x *IDs) Reset() {
	*x = IDs{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[8]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *IDs) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*IDs) ProtoMessage() {}

func (x *IDs) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use IDs.ProtoReflect.Descriptor instead.
func (*IDs) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{8}
}

func (x *IDs) GetIds() []*ProcessID {
	if x != nil {
		return x.Ids
	}
	return nil
}

type SignalRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Ids    []*ProcessID `protobuf:"bytes,1,rep,name=ids,proto3" json:"ids,omitempty"`
	Signal Signal       `protobuf:"varint,2,opt,name=signal,proto3,enum=api.Signal" json:"signal,omitempty"`
}

func (x *SignalRequest) Reset() {
	*x = SignalRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_api_api_proto_msgTypes[9]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *SignalRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*SignalRequest) ProtoMessage() {}

func (x *SignalRequest) ProtoReflect() protoreflect.Message {
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

// Deprecated: Use SignalRequest.ProtoReflect.Descriptor instead.
func (*SignalRequest) Descriptor() ([]byte, []int) {
	return file_api_api_proto_rawDescGZIP(), []int{9}
}

func (x *SignalRequest) GetIds() []*ProcessID {
	if x != nil {
		return x.Ids
	}
	return nil
}

func (x *SignalRequest) GetSignal() Signal {
	if x != nil {
		return x.Signal
	}
	return Signal_SIGNAL_UNSPECIFIED
}

var File_api_api_proto protoreflect.FileDescriptor

var file_api_api_proto_rawDesc = []byte{
	0x0a, 0x0d, 0x61, 0x70, 0x69, 0x2f, 0x61, 0x70, 0x69, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12,
	0x03, 0x61, 0x70, 0x69, 0x1a, 0x1b, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x62, 0x75, 0x66, 0x2f, 0x65, 0x6d, 0x70, 0x74, 0x79, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x1a, 0x1f, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2f, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2f, 0x74, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x22, 0x1b, 0x0a, 0x09, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x44, 0x12,
	0x0e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x04, 0x52, 0x02, 0x69, 0x64, 0x22,
	0x3e, 0x0a, 0x0d, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x12, 0x2d, 0x0a, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x13, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x4f,
	0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x52, 0x07, 0x6f, 0x70, 0x74, 0x69, 0x6f, 0x6e, 0x73, 0x22,
	0x86, 0x01, 0x0a, 0x0e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x4f, 0x70, 0x74, 0x69, 0x6f,
	0x6e, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e, 0x64, 0x12, 0x12, 0x0a, 0x04,
	0x61, 0x72, 0x67, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x61, 0x72, 0x67, 0x73,
	0x12, 0x17, 0x0a, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x48, 0x00,
	0x52, 0x04, 0x6e, 0x61, 0x6d, 0x65, 0x88, 0x01, 0x01, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x77, 0x64,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x63, 0x77, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x74,
	0x61, 0x67, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x42,
	0x07, 0x0a, 0x05, 0x5f, 0x6e, 0x61, 0x6d, 0x65, 0x22, 0x3b, 0x0a, 0x0d, 0x50, 0x72, 0x6f, 0x63,
	0x65, 0x73, 0x73, 0x65, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x2a, 0x0a, 0x09, 0x70, 0x72, 0x6f,
	0x63, 0x65, 0x73, 0x73, 0x65, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0c, 0x2e, 0x61,
	0x70, 0x69, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x52, 0x09, 0x70, 0x72, 0x6f, 0x63,
	0x65, 0x73, 0x73, 0x65, 0x73, 0x22, 0xbd, 0x01, 0x0a, 0x07, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73,
	0x73, 0x12, 0x1e, 0x0a, 0x02, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x44, 0x52, 0x02, 0x69,
	0x64, 0x12, 0x2a, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x18, 0x02, 0x20, 0x01, 0x28,
	0x0b, 0x32, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x53,
	0x74, 0x61, 0x74, 0x75, 0x73, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12, 0x0a,
	0x04, 0x6e, 0x61, 0x6d, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x6e, 0x61, 0x6d,
	0x65, 0x12, 0x10, 0x0a, 0x03, 0x63, 0x77, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03,
	0x63, 0x77, 0x64, 0x12, 0x12, 0x0a, 0x04, 0x74, 0x61, 0x67, 0x73, 0x18, 0x05, 0x20, 0x03, 0x28,
	0x09, 0x52, 0x04, 0x74, 0x61, 0x67, 0x73, 0x12, 0x18, 0x0a, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61,
	0x6e, 0x64, 0x18, 0x06, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x63, 0x6f, 0x6d, 0x6d, 0x61, 0x6e,
	0x64, 0x12, 0x12, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73, 0x18, 0x07, 0x20, 0x03, 0x28, 0x09, 0x52,
	0x04, 0x61, 0x72, 0x67, 0x73, 0x22, 0xf1, 0x01, 0x0a, 0x0d, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73,
	0x73, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x32, 0x0a, 0x07, 0x69, 0x6e, 0x76, 0x61, 0x6c,
	0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x48, 0x00, 0x52, 0x07, 0x69, 0x6e, 0x76, 0x61, 0x6c, 0x69, 0x64, 0x12, 0x34, 0x0a, 0x08, 0x73,
	0x74, 0x61, 0x72, 0x74, 0x69, 0x6e, 0x67, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x16, 0x2e,
	0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e,
	0x45, 0x6d, 0x70, 0x74, 0x79, 0x48, 0x00, 0x52, 0x08, 0x73, 0x74, 0x61, 0x72, 0x74, 0x69, 0x6e,
	0x67, 0x12, 0x35, 0x0a, 0x07, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x18, 0x03, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x19, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x52, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67,
	0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x53, 0x74, 0x61, 0x74, 0x75, 0x73, 0x48, 0x00, 0x52,
	0x07, 0x72, 0x75, 0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x12, 0x35, 0x0a, 0x07, 0x73, 0x74, 0x6f, 0x70,
	0x70, 0x65, 0x64, 0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x19, 0x2e, 0x61, 0x70, 0x69, 0x2e,
	0x53, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x53, 0x74,
	0x61, 0x74, 0x75, 0x73, 0x48, 0x00, 0x52, 0x07, 0x73, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x42,
	0x08, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x22, 0x8d, 0x01, 0x0a, 0x14, 0x52, 0x75,
	0x6e, 0x6e, 0x69, 0x6e, 0x67, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x53, 0x74, 0x61, 0x74,
	0x75, 0x73, 0x12, 0x10, 0x0a, 0x03, 0x70, 0x69, 0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x03, 0x52,
	0x03, 0x70, 0x69, 0x64, 0x12, 0x39, 0x0a, 0x0a, 0x73, 0x74, 0x61, 0x72, 0x74, 0x5f, 0x74, 0x69,
	0x6d, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73,
	0x74, 0x61, 0x6d, 0x70, 0x52, 0x09, 0x73, 0x74, 0x61, 0x72, 0x74, 0x54, 0x69, 0x6d, 0x65, 0x12,
	0x10, 0x0a, 0x03, 0x63, 0x70, 0x75, 0x18, 0x03, 0x20, 0x01, 0x28, 0x04, 0x52, 0x03, 0x63, 0x70,
	0x75, 0x12, 0x16, 0x0a, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x18, 0x04, 0x20, 0x01, 0x28,
	0x04, 0x52, 0x06, 0x6d, 0x65, 0x6d, 0x6f, 0x72, 0x79, 0x22, 0x6e, 0x0a, 0x14, 0x53, 0x74, 0x6f,
	0x70, 0x70, 0x65, 0x64, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x53, 0x74, 0x61, 0x74, 0x75,
	0x73, 0x12, 0x1b, 0x0a, 0x09, 0x65, 0x78, 0x69, 0x74, 0x5f, 0x63, 0x6f, 0x64, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x03, 0x52, 0x08, 0x65, 0x78, 0x69, 0x74, 0x43, 0x6f, 0x64, 0x65, 0x12, 0x39,
	0x0a, 0x0a, 0x73, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x5f, 0x61, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x0b, 0x32, 0x1a, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x54, 0x69, 0x6d, 0x65, 0x73, 0x74, 0x61, 0x6d, 0x70, 0x52, 0x09,
	0x73, 0x74, 0x6f, 0x70, 0x70, 0x65, 0x64, 0x41, 0x74, 0x22, 0x27, 0x0a, 0x03, 0x49, 0x44, 0x73,
	0x12, 0x20, 0x0a, 0x03, 0x69, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b, 0x32, 0x0e, 0x2e,
	0x61, 0x70, 0x69, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x44, 0x52, 0x03, 0x69,
	0x64, 0x73, 0x22, 0x56, 0x0a, 0x0d, 0x53, 0x69, 0x67, 0x6e, 0x61, 0x6c, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x20, 0x0a, 0x03, 0x69, 0x64, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x0b,
	0x32, 0x0e, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x50, 0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x49, 0x44,
	0x52, 0x03, 0x69, 0x64, 0x73, 0x12, 0x23, 0x0a, 0x06, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x6c, 0x18,
	0x02, 0x20, 0x01, 0x28, 0x0e, 0x32, 0x0b, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x69, 0x67, 0x6e,
	0x61, 0x6c, 0x52, 0x06, 0x73, 0x69, 0x67, 0x6e, 0x61, 0x6c, 0x2a, 0x48, 0x0a, 0x06, 0x53, 0x69,
	0x67, 0x6e, 0x61, 0x6c, 0x12, 0x16, 0x0a, 0x12, 0x53, 0x49, 0x47, 0x4e, 0x41, 0x4c, 0x5f, 0x55,
	0x4e, 0x53, 0x50, 0x45, 0x43, 0x49, 0x46, 0x49, 0x45, 0x44, 0x10, 0x00, 0x12, 0x12, 0x0a, 0x0e,
	0x53, 0x49, 0x47, 0x4e, 0x41, 0x4c, 0x5f, 0x53, 0x49, 0x47, 0x54, 0x45, 0x52, 0x4d, 0x10, 0x01,
	0x12, 0x12, 0x0a, 0x0e, 0x53, 0x49, 0x47, 0x4e, 0x41, 0x4c, 0x5f, 0x53, 0x49, 0x47, 0x4b, 0x49,
	0x4c, 0x4c, 0x10, 0x02, 0x32, 0xb0, 0x02, 0x0a, 0x06, 0x44, 0x61, 0x65, 0x6d, 0x6f, 0x6e, 0x12,
	0x29, 0x0a, 0x05, 0x53, 0x74, 0x61, 0x72, 0x74, 0x12, 0x08, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x49,
	0x44, 0x73, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74,
	0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x34, 0x0a, 0x06, 0x53, 0x69,
	0x67, 0x6e, 0x61, 0x6c, 0x12, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x53, 0x69, 0x67, 0x6e, 0x61,
	0x6c, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c,
	0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79,
	0x12, 0x26, 0x0a, 0x06, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x12, 0x12, 0x2e, 0x61, 0x70, 0x69,
	0x2e, 0x43, 0x72, 0x65, 0x61, 0x74, 0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x08,
	0x2e, 0x61, 0x70, 0x69, 0x2e, 0x49, 0x44, 0x73, 0x12, 0x32, 0x0a, 0x04, 0x4c, 0x69, 0x73, 0x74,
	0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a, 0x12, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x50,
	0x72, 0x6f, 0x63, 0x65, 0x73, 0x73, 0x65, 0x73, 0x4c, 0x69, 0x73, 0x74, 0x12, 0x2a, 0x0a, 0x06,
	0x44, 0x65, 0x6c, 0x65, 0x74, 0x65, 0x12, 0x08, 0x2e, 0x61, 0x70, 0x69, 0x2e, 0x49, 0x44, 0x73,
	0x1a, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62,
	0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x12, 0x3d, 0x0a, 0x0b, 0x48, 0x65, 0x61, 0x6c,
	0x74, 0x68, 0x43, 0x68, 0x65, 0x63, 0x6b, 0x12, 0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75, 0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x1a,
	0x16, 0x2e, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x62, 0x75,
	0x66, 0x2e, 0x45, 0x6d, 0x70, 0x74, 0x79, 0x42, 0x1c, 0x5a, 0x1a, 0x67, 0x69, 0x74, 0x68, 0x75,
	0x62, 0x2e, 0x63, 0x6f, 0x6d, 0x2f, 0x72, 0x70, 0x72, 0x74, 0x72, 0x32, 0x35, 0x38, 0x2f, 0x70,
	0x6d, 0x2f, 0x61, 0x70, 0x69, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
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

var file_api_api_proto_enumTypes = make([]protoimpl.EnumInfo, 1)
var file_api_api_proto_msgTypes = make([]protoimpl.MessageInfo, 10)
var file_api_api_proto_goTypes = []interface{}{
	(Signal)(0),                   // 0: api.Signal
	(*ProcessID)(nil),             // 1: api.ProcessID
	(*CreateRequest)(nil),         // 2: api.CreateRequest
	(*ProcessOptions)(nil),        // 3: api.ProcessOptions
	(*ProcessesList)(nil),         // 4: api.ProcessesList
	(*Process)(nil),               // 5: api.Process
	(*ProcessStatus)(nil),         // 6: api.ProcessStatus
	(*RunningProcessStatus)(nil),  // 7: api.RunningProcessStatus
	(*StoppedProcessStatus)(nil),  // 8: api.StoppedProcessStatus
	(*IDs)(nil),                   // 9: api.IDs
	(*SignalRequest)(nil),         // 10: api.SignalRequest
	(*emptypb.Empty)(nil),         // 11: google.protobuf.Empty
	(*timestamppb.Timestamp)(nil), // 12: google.protobuf.Timestamp
}
var file_api_api_proto_depIdxs = []int32{
	3,  // 0: api.CreateRequest.options:type_name -> api.ProcessOptions
	5,  // 1: api.ProcessesList.processes:type_name -> api.Process
	1,  // 2: api.Process.id:type_name -> api.ProcessID
	6,  // 3: api.Process.status:type_name -> api.ProcessStatus
	11, // 4: api.ProcessStatus.invalid:type_name -> google.protobuf.Empty
	11, // 5: api.ProcessStatus.starting:type_name -> google.protobuf.Empty
	7,  // 6: api.ProcessStatus.running:type_name -> api.RunningProcessStatus
	8,  // 7: api.ProcessStatus.stopped:type_name -> api.StoppedProcessStatus
	12, // 8: api.RunningProcessStatus.start_time:type_name -> google.protobuf.Timestamp
	12, // 9: api.StoppedProcessStatus.stopped_at:type_name -> google.protobuf.Timestamp
	1,  // 10: api.IDs.ids:type_name -> api.ProcessID
	1,  // 11: api.SignalRequest.ids:type_name -> api.ProcessID
	0,  // 12: api.SignalRequest.signal:type_name -> api.Signal
	9,  // 13: api.Daemon.Start:input_type -> api.IDs
	10, // 14: api.Daemon.Signal:input_type -> api.SignalRequest
	2,  // 15: api.Daemon.Create:input_type -> api.CreateRequest
	11, // 16: api.Daemon.List:input_type -> google.protobuf.Empty
	9,  // 17: api.Daemon.Delete:input_type -> api.IDs
	11, // 18: api.Daemon.HealthCheck:input_type -> google.protobuf.Empty
	11, // 19: api.Daemon.Start:output_type -> google.protobuf.Empty
	11, // 20: api.Daemon.Signal:output_type -> google.protobuf.Empty
	9,  // 21: api.Daemon.Create:output_type -> api.IDs
	4,  // 22: api.Daemon.List:output_type -> api.ProcessesList
	11, // 23: api.Daemon.Delete:output_type -> google.protobuf.Empty
	11, // 24: api.Daemon.HealthCheck:output_type -> google.protobuf.Empty
	19, // [19:25] is the sub-list for method output_type
	13, // [13:19] is the sub-list for method input_type
	13, // [13:13] is the sub-list for extension type_name
	13, // [13:13] is the sub-list for extension extendee
	0,  // [0:13] is the sub-list for field type_name
}

func init() { file_api_api_proto_init() }
func file_api_api_proto_init() {
	if File_api_api_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_api_api_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ProcessID); i {
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
			switch v := v.(*CreateRequest); i {
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
			switch v := v.(*ProcessOptions); i {
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
			switch v := v.(*ProcessesList); i {
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
			switch v := v.(*Process); i {
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
			switch v := v.(*ProcessStatus); i {
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
			switch v := v.(*RunningProcessStatus); i {
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
			switch v := v.(*StoppedProcessStatus); i {
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
			switch v := v.(*IDs); i {
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
			switch v := v.(*SignalRequest); i {
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
	file_api_api_proto_msgTypes[2].OneofWrappers = []interface{}{}
	file_api_api_proto_msgTypes[5].OneofWrappers = []interface{}{
		(*ProcessStatus_Invalid)(nil),
		(*ProcessStatus_Starting)(nil),
		(*ProcessStatus_Running)(nil),
		(*ProcessStatus_Stopped)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_api_api_proto_rawDesc,
			NumEnums:      1,
			NumMessages:   10,
			NumExtensions: 0,
			NumServices:   1,
		},
		GoTypes:           file_api_api_proto_goTypes,
		DependencyIndexes: file_api_api_proto_depIdxs,
		EnumInfos:         file_api_api_proto_enumTypes,
		MessageInfos:      file_api_api_proto_msgTypes,
	}.Build()
	File_api_api_proto = out.File
	file_api_api_proto_rawDesc = nil
	file_api_api_proto_goTypes = nil
	file_api_api_proto_depIdxs = nil
}
