// Code generated by protoc-gen-go.
// source: hapi/release/status.proto
// DO NOT EDIT!

package release

import proto "github.com/golang/protobuf/proto"
import fmt "fmt"
import math "math"
import google_protobuf1 "github.com/golang/protobuf/ptypes/any"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = fmt.Errorf
var _ = math.Inf

type Status_Code int32

const (
	// Status_UNKNOWN indicates that a release is in an uncertain state.
	Status_UNKNOWN Status_Code = 0
	// Status_DEPLOYED indicates that the release has been pushed to Kubernetes.
	Status_DEPLOYED Status_Code = 1
	// Status_DELETED indicates that a release has been deleted from Kubermetes.
	Status_DELETED Status_Code = 2
	// Status_SUPERSEDED indicates that this release object is outdated and a newer one exists.
	Status_SUPERSEDED Status_Code = 3
	// Status_FAILED indicates that the release was not successfully deployed.
	Status_FAILED Status_Code = 4
	// Status_DELETING indicates that a delete operation is underway.
	Status_DELETING Status_Code = 5
)

var Status_Code_name = map[int32]string{
	0: "UNKNOWN",
	1: "DEPLOYED",
	2: "DELETED",
	3: "SUPERSEDED",
	4: "FAILED",
	5: "DELETING",
}
var Status_Code_value = map[string]int32{
	"UNKNOWN":    0,
	"DEPLOYED":   1,
	"DELETED":    2,
	"SUPERSEDED": 3,
	"FAILED":     4,
	"DELETING":   5,
}

func (x Status_Code) String() string {
	return proto.EnumName(Status_Code_name, int32(x))
}
func (Status_Code) EnumDescriptor() ([]byte, []int) { return fileDescriptor3, []int{0, 0} }

// Status defines the status of a release.
type Status struct {
	Code    Status_Code           `protobuf:"varint,1,opt,name=code,enum=hapi.release.Status_Code" json:"code,omitempty"`
	Details *google_protobuf1.Any `protobuf:"bytes,2,opt,name=details" json:"details,omitempty"`
	// Cluster resources as kubectl would print them.
	Resources string `protobuf:"bytes,3,opt,name=resources" json:"resources,omitempty"`
	// Contains the rendered templates/NOTES.txt if available
	Notes string `protobuf:"bytes,4,opt,name=notes" json:"notes,omitempty"`
}

func (m *Status) Reset()                    { *m = Status{} }
func (m *Status) String() string            { return proto.CompactTextString(m) }
func (*Status) ProtoMessage()               {}
func (*Status) Descriptor() ([]byte, []int) { return fileDescriptor3, []int{0} }

func (m *Status) GetDetails() *google_protobuf1.Any {
	if m != nil {
		return m.Details
	}
	return nil
}

func init() {
	proto.RegisterType((*Status)(nil), "hapi.release.Status")
	proto.RegisterEnum("hapi.release.Status_Code", Status_Code_name, Status_Code_value)
}

func init() { proto.RegisterFile("hapi/release/status.proto", fileDescriptor3) }

var fileDescriptor3 = []byte{
	// 269 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x09, 0x6e, 0x88, 0x02, 0xff, 0x4c, 0x8f, 0x4d, 0x6f, 0x82, 0x40,
	0x10, 0x86, 0xbb, 0x8a, 0x50, 0x46, 0x63, 0x36, 0x1b, 0x0f, 0xd0, 0xf4, 0x40, 0x3c, 0x71, 0xe9,
	0x92, 0xd8, 0x5f, 0x60, 0xbb, 0xdb, 0xc6, 0x94, 0xa0, 0x01, 0x4d, 0x3f, 0x6e, 0x28, 0x53, 0x6b,
	0x42, 0x58, 0xc3, 0xc2, 0xc1, 0x1f, 0xde, 0x7b, 0x03, 0x68, 0xea, 0x71, 0xf7, 0x79, 0xde, 0x79,
	0x67, 0xc0, 0xfd, 0x49, 0x8f, 0x87, 0xa0, 0xc4, 0x1c, 0x53, 0x8d, 0x81, 0xae, 0xd2, 0xaa, 0xd6,
	0xfc, 0x58, 0xaa, 0x4a, 0xb1, 0x51, 0x83, 0xf8, 0x19, 0xdd, 0xb9, 0x7b, 0xa5, 0xf6, 0x39, 0x06,
	0x2d, 0xdb, 0xd6, 0xdf, 0x41, 0x5a, 0x9c, 0x3a, 0x71, 0xfa, 0x4b, 0xc0, 0x4c, 0xda, 0x24, 0x7b,
	0x00, 0x63, 0xa7, 0x32, 0x74, 0x88, 0x47, 0xfc, 0xf1, 0xcc, 0xe5, 0xd7, 0x23, 0x78, 0xe7, 0xf0,
	0x67, 0x95, 0x61, 0xdc, 0x6a, 0x8c, 0x83, 0x95, 0x61, 0x95, 0x1e, 0x72, 0xed, 0xf4, 0x3c, 0xe2,
	0x0f, 0x67, 0x13, 0xde, 0xd5, 0xf0, 0x4b, 0x0d, 0x9f, 0x17, 0xa7, 0xf8, 0x22, 0xb1, 0x7b, 0xb0,
	0x4b, 0xd4, 0xaa, 0x2e, 0x77, 0xa8, 0x9d, 0xbe, 0x47, 0x7c, 0x3b, 0xfe, 0xff, 0x60, 0x13, 0x18,
	0x14, 0xaa, 0x42, 0xed, 0x18, 0x2d, 0xe9, 0x1e, 0xd3, 0x0f, 0x30, 0x9a, 0x46, 0x36, 0x04, 0x6b,
	0x13, 0xbd, 0x45, 0xcb, 0xf7, 0x88, 0xde, 0xb0, 0x11, 0xdc, 0x0a, 0xb9, 0x0a, 0x97, 0x9f, 0x52,
	0x50, 0xd2, 0x20, 0x21, 0x43, 0xb9, 0x96, 0x82, 0xf6, 0xd8, 0x18, 0x20, 0xd9, 0xac, 0x64, 0x9c,
	0x48, 0x21, 0x05, 0xed, 0x33, 0x00, 0xf3, 0x65, 0xbe, 0x08, 0xa5, 0xa0, 0x46, 0x17, 0x0b, 0xe5,
	0x7a, 0x11, 0xbd, 0xd2, 0xc1, 0x93, 0xfd, 0x65, 0x9d, 0x4f, 0xdb, 0x9a, 0xed, 0xbe, 0x8f, 0x7f,
	0x01, 0x00, 0x00, 0xff, 0xff, 0xc8, 0x7b, 0x5f, 0x3b, 0x4f, 0x01, 0x00, 0x00,
}
