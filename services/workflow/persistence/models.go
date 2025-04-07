package persistence

import (
	pb "paul.hobbs.page/aisociety/protos"
)

// TasksCollection is a protobuf-compatible wrapper for a collection of Task messages
type TasksCollection struct {
	Tasks []*pb.Task `protobuf:"bytes,1,rep,name=tasks,proto3" json:"tasks,omitempty"`
}

// Reset implements proto.Message
func (x *TasksCollection) Reset() {
	*x = TasksCollection{}
}

// String implements proto.Message
func (x *TasksCollection) String() string {
	return "TasksCollection"
}

// ProtoMessage implements proto.Message
func (*TasksCollection) ProtoMessage() {}

// EditsCollection is a protobuf-compatible wrapper for a collection of NodeEdit messages
type EditsCollection struct {
	Edits []*pb.NodeEdit `protobuf:"bytes,1,rep,name=edits,proto3" json:"edits,omitempty"`
}

// Reset implements proto.Message
func (x *EditsCollection) Reset() {
	*x = EditsCollection{}
}

// String implements proto.Message
func (x *EditsCollection) String() string {
	return "EditsCollection"
}

// ProtoMessage implements proto.Message
func (*EditsCollection) ProtoMessage() {}
