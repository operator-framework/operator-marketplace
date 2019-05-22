// Code generated by protoc-gen-go. DO NOT EDIT.
// source: grafeas/v1/package.proto

package grafeas

import (
	fmt "fmt"
	math "math"

	proto "github.com/golang/protobuf/proto"
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

// Instruction set architectures supported by various package managers.
type Architecture int32

const (
	// Unknown architecture.
	Architecture_ARCHITECTURE_UNSPECIFIED Architecture = 0
	// X86 architecture.
	Architecture_X86 Architecture = 1
	// X64 architecture.
	Architecture_X64 Architecture = 2
)

var Architecture_name = map[int32]string{
	0: "ARCHITECTURE_UNSPECIFIED",
	1: "X86",
	2: "X64",
}

var Architecture_value = map[string]int32{
	"ARCHITECTURE_UNSPECIFIED": 0,
	"X86":                      1,
	"X64":                      2,
}

func (x Architecture) String() string {
	return proto.EnumName(Architecture_name, int32(x))
}

func (Architecture) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{0}
}

// Whether this is an ordinary package version or a sentinel MIN/MAX version.
type Version_VersionKind int32

const (
	// Unknown.
	Version_VERSION_KIND_UNSPECIFIED Version_VersionKind = 0
	// A standard package version.
	Version_NORMAL Version_VersionKind = 1
	// A special version representing negative infinity.
	Version_MINIMUM Version_VersionKind = 2
	// A special version representing positive infinity.
	Version_MAXIMUM Version_VersionKind = 3
)

var Version_VersionKind_name = map[int32]string{
	0: "VERSION_KIND_UNSPECIFIED",
	1: "NORMAL",
	2: "MINIMUM",
	3: "MAXIMUM",
}

var Version_VersionKind_value = map[string]int32{
	"VERSION_KIND_UNSPECIFIED": 0,
	"NORMAL":                   1,
	"MINIMUM":                  2,
	"MAXIMUM":                  3,
}

func (x Version_VersionKind) String() string {
	return proto.EnumName(Version_VersionKind_name, int32(x))
}

func (Version_VersionKind) EnumDescriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{5, 0}
}

// This represents a particular channel of distribution for a given package.
// E.g., Debian's jessie-backports dpkg mirror.
type Distribution struct {
	// Required. The cpe_uri in [CPE format](https://cpe.mitre.org/specification/)
	// denoting the package manager version distributing a package.
	CpeUri string `protobuf:"bytes,1,opt,name=cpe_uri,json=cpeUri,proto3" json:"cpe_uri,omitempty"`
	// The CPU architecture for which packages in this distribution channel were
	// built.
	Architecture Architecture `protobuf:"varint,2,opt,name=architecture,proto3,enum=grafeas.v1.Architecture" json:"architecture,omitempty"`
	// The latest available version of this package in this distribution channel.
	LatestVersion *Version `protobuf:"bytes,3,opt,name=latest_version,json=latestVersion,proto3" json:"latest_version,omitempty"`
	// A freeform string denoting the maintainer of this package.
	Maintainer string `protobuf:"bytes,4,opt,name=maintainer,proto3" json:"maintainer,omitempty"`
	// The distribution channel-specific homepage for this package.
	Url string `protobuf:"bytes,5,opt,name=url,proto3" json:"url,omitempty"`
	// The distribution channel-specific description of this package.
	Description          string   `protobuf:"bytes,6,opt,name=description,proto3" json:"description,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Distribution) Reset()         { *m = Distribution{} }
func (m *Distribution) String() string { return proto.CompactTextString(m) }
func (*Distribution) ProtoMessage()    {}
func (*Distribution) Descriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{0}
}

func (m *Distribution) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Distribution.Unmarshal(m, b)
}
func (m *Distribution) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Distribution.Marshal(b, m, deterministic)
}
func (m *Distribution) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Distribution.Merge(m, src)
}
func (m *Distribution) XXX_Size() int {
	return xxx_messageInfo_Distribution.Size(m)
}
func (m *Distribution) XXX_DiscardUnknown() {
	xxx_messageInfo_Distribution.DiscardUnknown(m)
}

var xxx_messageInfo_Distribution proto.InternalMessageInfo

func (m *Distribution) GetCpeUri() string {
	if m != nil {
		return m.CpeUri
	}
	return ""
}

func (m *Distribution) GetArchitecture() Architecture {
	if m != nil {
		return m.Architecture
	}
	return Architecture_ARCHITECTURE_UNSPECIFIED
}

func (m *Distribution) GetLatestVersion() *Version {
	if m != nil {
		return m.LatestVersion
	}
	return nil
}

func (m *Distribution) GetMaintainer() string {
	if m != nil {
		return m.Maintainer
	}
	return ""
}

func (m *Distribution) GetUrl() string {
	if m != nil {
		return m.Url
	}
	return ""
}

func (m *Distribution) GetDescription() string {
	if m != nil {
		return m.Description
	}
	return ""
}

// An occurrence of a particular package installation found within a system's
// filesystem. E.g., glibc was found in `/var/lib/dpkg/status`.
type Location struct {
	// Required. The CPE URI in [CPE format](https://cpe.mitre.org/specification/)
	// denoting the package manager version distributing a package.
	CpeUri string `protobuf:"bytes,1,opt,name=cpe_uri,json=cpeUri,proto3" json:"cpe_uri,omitempty"`
	// The version installed at this location.
	Version *Version `protobuf:"bytes,2,opt,name=version,proto3" json:"version,omitempty"`
	// The path from which we gathered that this package/version is installed.
	Path                 string   `protobuf:"bytes,3,opt,name=path,proto3" json:"path,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Location) Reset()         { *m = Location{} }
func (m *Location) String() string { return proto.CompactTextString(m) }
func (*Location) ProtoMessage()    {}
func (*Location) Descriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{1}
}

func (m *Location) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Location.Unmarshal(m, b)
}
func (m *Location) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Location.Marshal(b, m, deterministic)
}
func (m *Location) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Location.Merge(m, src)
}
func (m *Location) XXX_Size() int {
	return xxx_messageInfo_Location.Size(m)
}
func (m *Location) XXX_DiscardUnknown() {
	xxx_messageInfo_Location.DiscardUnknown(m)
}

var xxx_messageInfo_Location proto.InternalMessageInfo

func (m *Location) GetCpeUri() string {
	if m != nil {
		return m.CpeUri
	}
	return ""
}

func (m *Location) GetVersion() *Version {
	if m != nil {
		return m.Version
	}
	return nil
}

func (m *Location) GetPath() string {
	if m != nil {
		return m.Path
	}
	return ""
}

// This represents a particular package that is distributed over various
// channels. E.g., glibc (aka libc6) is distributed by many, at various
// versions.
type PackageNote struct {
	// Required. Immutable. The name of the package.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// The various channels by which a package is distributed.
	Distribution         []*Distribution `protobuf:"bytes,10,rep,name=distribution,proto3" json:"distribution,omitempty"`
	XXX_NoUnkeyedLiteral struct{}        `json:"-"`
	XXX_unrecognized     []byte          `json:"-"`
	XXX_sizecache        int32           `json:"-"`
}

func (m *PackageNote) Reset()         { *m = PackageNote{} }
func (m *PackageNote) String() string { return proto.CompactTextString(m) }
func (*PackageNote) ProtoMessage()    {}
func (*PackageNote) Descriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{2}
}

func (m *PackageNote) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PackageNote.Unmarshal(m, b)
}
func (m *PackageNote) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PackageNote.Marshal(b, m, deterministic)
}
func (m *PackageNote) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PackageNote.Merge(m, src)
}
func (m *PackageNote) XXX_Size() int {
	return xxx_messageInfo_PackageNote.Size(m)
}
func (m *PackageNote) XXX_DiscardUnknown() {
	xxx_messageInfo_PackageNote.DiscardUnknown(m)
}

var xxx_messageInfo_PackageNote proto.InternalMessageInfo

func (m *PackageNote) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *PackageNote) GetDistribution() []*Distribution {
	if m != nil {
		return m.Distribution
	}
	return nil
}

// Details of a package occurrence.
type PackageOccurrence struct {
	// Required. Where the package was installed.
	Installation         *Installation `protobuf:"bytes,1,opt,name=installation,proto3" json:"installation,omitempty"`
	XXX_NoUnkeyedLiteral struct{}      `json:"-"`
	XXX_unrecognized     []byte        `json:"-"`
	XXX_sizecache        int32         `json:"-"`
}

func (m *PackageOccurrence) Reset()         { *m = PackageOccurrence{} }
func (m *PackageOccurrence) String() string { return proto.CompactTextString(m) }
func (*PackageOccurrence) ProtoMessage()    {}
func (*PackageOccurrence) Descriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{3}
}

func (m *PackageOccurrence) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_PackageOccurrence.Unmarshal(m, b)
}
func (m *PackageOccurrence) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_PackageOccurrence.Marshal(b, m, deterministic)
}
func (m *PackageOccurrence) XXX_Merge(src proto.Message) {
	xxx_messageInfo_PackageOccurrence.Merge(m, src)
}
func (m *PackageOccurrence) XXX_Size() int {
	return xxx_messageInfo_PackageOccurrence.Size(m)
}
func (m *PackageOccurrence) XXX_DiscardUnknown() {
	xxx_messageInfo_PackageOccurrence.DiscardUnknown(m)
}

var xxx_messageInfo_PackageOccurrence proto.InternalMessageInfo

func (m *PackageOccurrence) GetInstallation() *Installation {
	if m != nil {
		return m.Installation
	}
	return nil
}

// This represents how a particular software package may be installed on a
// system.
type Installation struct {
	// Output only. The name of the installed package.
	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// Required. All of the places within the filesystem versions of this package
	// have been found.
	Location             []*Location `protobuf:"bytes,2,rep,name=location,proto3" json:"location,omitempty"`
	XXX_NoUnkeyedLiteral struct{}    `json:"-"`
	XXX_unrecognized     []byte      `json:"-"`
	XXX_sizecache        int32       `json:"-"`
}

func (m *Installation) Reset()         { *m = Installation{} }
func (m *Installation) String() string { return proto.CompactTextString(m) }
func (*Installation) ProtoMessage()    {}
func (*Installation) Descriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{4}
}

func (m *Installation) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Installation.Unmarshal(m, b)
}
func (m *Installation) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Installation.Marshal(b, m, deterministic)
}
func (m *Installation) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Installation.Merge(m, src)
}
func (m *Installation) XXX_Size() int {
	return xxx_messageInfo_Installation.Size(m)
}
func (m *Installation) XXX_DiscardUnknown() {
	xxx_messageInfo_Installation.DiscardUnknown(m)
}

var xxx_messageInfo_Installation proto.InternalMessageInfo

func (m *Installation) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Installation) GetLocation() []*Location {
	if m != nil {
		return m.Location
	}
	return nil
}

// Version contains structured information about the version of a package.
type Version struct {
	// Used to correct mistakes in the version numbering scheme.
	Epoch int32 `protobuf:"varint,1,opt,name=epoch,proto3" json:"epoch,omitempty"`
	// Required only when version kind is NORMAL. The main part of the version
	// name.
	Name string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	// The iteration of the package build from the above version.
	Revision string `protobuf:"bytes,3,opt,name=revision,proto3" json:"revision,omitempty"`
	// Required. Distinguishes between sentinel MIN/MAX versions and normal
	// versions.
	Kind Version_VersionKind `protobuf:"varint,4,opt,name=kind,proto3,enum=grafeas.v1.Version_VersionKind" json:"kind,omitempty"`
	// Human readable version string. This string is of the form
	// <epoch>:<name>-<revision> and is only set when kind is NORMAL.
	FullName             string   `protobuf:"bytes,5,opt,name=full_name,json=fullName,proto3" json:"full_name,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}

func (m *Version) Reset()         { *m = Version{} }
func (m *Version) String() string { return proto.CompactTextString(m) }
func (*Version) ProtoMessage()    {}
func (*Version) Descriptor() ([]byte, []int) {
	return fileDescriptor_6152b3fff9015bb3, []int{5}
}

func (m *Version) XXX_Unmarshal(b []byte) error {
	return xxx_messageInfo_Version.Unmarshal(m, b)
}
func (m *Version) XXX_Marshal(b []byte, deterministic bool) ([]byte, error) {
	return xxx_messageInfo_Version.Marshal(b, m, deterministic)
}
func (m *Version) XXX_Merge(src proto.Message) {
	xxx_messageInfo_Version.Merge(m, src)
}
func (m *Version) XXX_Size() int {
	return xxx_messageInfo_Version.Size(m)
}
func (m *Version) XXX_DiscardUnknown() {
	xxx_messageInfo_Version.DiscardUnknown(m)
}

var xxx_messageInfo_Version proto.InternalMessageInfo

func (m *Version) GetEpoch() int32 {
	if m != nil {
		return m.Epoch
	}
	return 0
}

func (m *Version) GetName() string {
	if m != nil {
		return m.Name
	}
	return ""
}

func (m *Version) GetRevision() string {
	if m != nil {
		return m.Revision
	}
	return ""
}

func (m *Version) GetKind() Version_VersionKind {
	if m != nil {
		return m.Kind
	}
	return Version_VERSION_KIND_UNSPECIFIED
}

func (m *Version) GetFullName() string {
	if m != nil {
		return m.FullName
	}
	return ""
}

func init() {
	proto.RegisterEnum("grafeas.v1.Architecture", Architecture_name, Architecture_value)
	proto.RegisterEnum("grafeas.v1.Version_VersionKind", Version_VersionKind_name, Version_VersionKind_value)
	proto.RegisterType((*Distribution)(nil), "grafeas.v1.Distribution")
	proto.RegisterType((*Location)(nil), "grafeas.v1.Location")
	proto.RegisterType((*PackageNote)(nil), "grafeas.v1.PackageNote")
	proto.RegisterType((*PackageOccurrence)(nil), "grafeas.v1.PackageOccurrence")
	proto.RegisterType((*Installation)(nil), "grafeas.v1.Installation")
	proto.RegisterType((*Version)(nil), "grafeas.v1.Version")
}

func init() { proto.RegisterFile("grafeas/v1/package.proto", fileDescriptor_6152b3fff9015bb3) }

var fileDescriptor_6152b3fff9015bb3 = []byte{
	// 562 bytes of a gzipped FileDescriptorProto
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0xff, 0x7c, 0x53, 0x5d, 0x6f, 0xd3, 0x30,
	0x14, 0x25, 0xc9, 0xd6, 0x8f, 0x9b, 0x6e, 0x0a, 0x66, 0x12, 0x11, 0x20, 0xa8, 0xf2, 0x54, 0x21,
	0x91, 0xb2, 0x0d, 0x4d, 0x13, 0x20, 0xa4, 0xb2, 0x15, 0x88, 0xb6, 0x66, 0x9b, 0xb7, 0x4e, 0x13,
	0x2f, 0x91, 0x97, 0xba, 0xa9, 0xb5, 0x34, 0x8e, 0x9c, 0xa4, 0x3f, 0x88, 0xdf, 0x08, 0xef, 0x28,
	0x4e, 0xda, 0xb9, 0xd3, 0xe0, 0x29, 0xf7, 0xfa, 0x9c, 0x9c, 0x73, 0x3f, 0x6c, 0xb0, 0x23, 0x41,
	0xa6, 0x94, 0x64, 0xfd, 0xc5, 0x6e, 0x3f, 0x25, 0xe1, 0x1d, 0x89, 0xa8, 0x9b, 0x0a, 0x9e, 0x73,
	0x04, 0x35, 0xe2, 0x2e, 0x76, 0x9d, 0x3f, 0x1a, 0x74, 0x8e, 0x59, 0x96, 0x0b, 0x76, 0x5b, 0xe4,
	0x8c, 0x27, 0xe8, 0x39, 0x34, 0xc3, 0x94, 0x06, 0x85, 0x60, 0xb6, 0xd6, 0xd5, 0x7a, 0x6d, 0xdc,
	0x08, 0x53, 0x3a, 0x16, 0x0c, 0x7d, 0x86, 0x0e, 0x11, 0xe1, 0x8c, 0xe5, 0x34, 0xcc, 0x0b, 0x41,
	0x6d, 0xbd, 0xab, 0xf5, 0xb6, 0xf7, 0x6c, 0xf7, 0x5e, 0xcc, 0x1d, 0x28, 0x38, 0x5e, 0x63, 0xa3,
	0x8f, 0xb0, 0x1d, 0x93, 0x9c, 0x66, 0x79, 0xb0, 0xa0, 0x22, 0x63, 0x3c, 0xb1, 0x8d, 0xae, 0xd6,
	0x33, 0xf7, 0x9e, 0xa9, 0xff, 0x5f, 0x57, 0x10, 0xde, 0xaa, 0xa8, 0x75, 0x8a, 0x5e, 0x03, 0xcc,
	0x09, 0x4b, 0x72, 0xc2, 0x12, 0x2a, 0xec, 0x0d, 0x59, 0x95, 0x72, 0x82, 0x2c, 0x30, 0x0a, 0x11,
	0xdb, 0x9b, 0x12, 0x28, 0x43, 0xd4, 0x05, 0x73, 0x42, 0xb3, 0x50, 0xb0, 0xb4, 0xec, 0xc9, 0x6e,
	0x48, 0x44, 0x3d, 0x72, 0xa6, 0xd0, 0x3a, 0xe5, 0x21, 0xf9, 0x7f, 0xcb, 0xef, 0xa0, 0xb9, 0xac,
	0x56, 0xff, 0x77, 0xb5, 0x4b, 0x0e, 0x42, 0xb0, 0x91, 0x92, 0x7c, 0x26, 0x3b, 0x6b, 0x63, 0x19,
	0x3b, 0x01, 0x98, 0xe7, 0xd5, 0xf0, 0x7d, 0x9e, 0xd3, 0x92, 0x92, 0x90, 0x39, 0xad, 0x7d, 0x64,
	0x5c, 0x0e, 0x76, 0xa2, 0x6c, 0xc0, 0x86, 0xae, 0xd1, 0x33, 0xd7, 0x07, 0xab, 0x6e, 0x08, 0xaf,
	0xb1, 0x9d, 0x0b, 0x78, 0x5a, 0x1b, 0x9c, 0x85, 0x61, 0x21, 0x04, 0x4d, 0x42, 0x29, 0xc9, 0x92,
	0x2c, 0x27, 0x71, 0x2c, 0x3b, 0x94, 0x76, 0x0f, 0x24, 0x3d, 0x05, 0xc7, 0x6b, 0x6c, 0xe7, 0x0a,
	0x3a, 0x2a, 0xfa, 0x68, 0xd1, 0xef, 0xa1, 0x15, 0xd7, 0xf3, 0xb3, 0x75, 0x59, 0xf0, 0x8e, 0xaa,
	0xbe, 0x9c, 0x2d, 0x5e, 0xb1, 0x9c, 0xdf, 0x1a, 0x34, 0x97, 0x1b, 0xdd, 0x81, 0x4d, 0x9a, 0xf2,
	0x70, 0x26, 0x25, 0x37, 0x71, 0x95, 0xac, 0x7c, 0x74, 0xc5, 0xe7, 0x05, 0xb4, 0x04, 0x5d, 0xb0,
	0xd5, 0x8d, 0x69, 0xe3, 0x55, 0x8e, 0xf6, 0x61, 0xe3, 0x8e, 0x25, 0x13, 0x79, 0x23, 0xb6, 0xf7,
	0xde, 0x3c, 0xb2, 0x9b, 0xe5, 0xf7, 0x84, 0x25, 0x13, 0x2c, 0xc9, 0xe8, 0x25, 0xb4, 0xa7, 0x45,
	0x1c, 0x07, 0xd2, 0xa9, 0xba, 0x32, 0xad, 0xf2, 0xc0, 0x27, 0x73, 0xea, 0x5c, 0x80, 0xa9, 0xfc,
	0x81, 0x5e, 0x81, 0x7d, 0x3d, 0xc4, 0x97, 0xde, 0x99, 0x1f, 0x9c, 0x78, 0xfe, 0x71, 0x30, 0xf6,
	0x2f, 0xcf, 0x87, 0x47, 0xde, 0x37, 0x6f, 0x78, 0x6c, 0x3d, 0x41, 0x00, 0x0d, 0xff, 0x0c, 0x8f,
	0x06, 0xa7, 0x96, 0x86, 0x4c, 0x68, 0x8e, 0x3c, 0xdf, 0x1b, 0x8d, 0x47, 0x96, 0x2e, 0x93, 0xc1,
	0x8d, 0x4c, 0x8c, 0xb7, 0x5f, 0xa0, 0xa3, 0x3e, 0x8b, 0x52, 0x73, 0x80, 0x8f, 0x7e, 0x78, 0x57,
	0xc3, 0xa3, 0xab, 0x31, 0x1e, 0x3e, 0xd0, 0x6c, 0x82, 0x71, 0x73, 0x78, 0x60, 0x69, 0x32, 0x38,
	0xf8, 0x60, 0xe9, 0x5f, 0x2f, 0x60, 0x8b, 0x71, 0xa5, 0xb5, 0x73, 0xed, 0xe7, 0x61, 0xc4, 0x79,
	0x14, 0x53, 0x37, 0xe2, 0x31, 0x49, 0x22, 0x97, 0x8b, 0xa8, 0x1f, 0xd1, 0x44, 0xbe, 0xed, 0x7e,
	0x05, 0x91, 0x94, 0x65, 0xfd, 0xfb, 0xf7, 0xff, 0xa9, 0x0e, 0x7f, 0xe9, 0xc6, 0x77, 0x3c, 0xb8,
	0x6d, 0x48, 0xea, 0xfe, 0xdf, 0x00, 0x00, 0x00, 0xff, 0xff, 0xae, 0x5f, 0x88, 0x6f, 0x22, 0x04,
	0x00, 0x00,
}