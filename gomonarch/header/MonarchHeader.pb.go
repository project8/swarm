// Code generated by protoc-gen-go.
// source: MonarchHeader.proto
// DO NOT EDIT!

package Protobuf

import proto "code.google.com/p/goprotobuf/proto"
import json "encoding/json"
import math "math"

// Reference proto, json, and math imports to suppress error if they are not otherwise used.
var _ = proto.Marshal
var _ = &json.SyntaxError{}
var _ = math.Inf

type MonarchHeader_RunSource int32

const (
	MonarchHeader_Mantis     MonarchHeader_RunSource = 0
	MonarchHeader_Simulation MonarchHeader_RunSource = 1
)

var MonarchHeader_RunSource_name = map[int32]string{
	0: "Mantis",
	1: "Simulation",
}
var MonarchHeader_RunSource_value = map[string]int32{
	"Mantis":     0,
	"Simulation": 1,
}

func (x MonarchHeader_RunSource) Enum() *MonarchHeader_RunSource {
	p := new(MonarchHeader_RunSource)
	*p = x
	return p
}
func (x MonarchHeader_RunSource) String() string {
	return proto.EnumName(MonarchHeader_RunSource_name, int32(x))
}
func (x MonarchHeader_RunSource) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
func (x *MonarchHeader_RunSource) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(MonarchHeader_RunSource_value, data, "MonarchHeader_RunSource")
	if err != nil {
		return err
	}
	*x = MonarchHeader_RunSource(value)
	return nil
}

type MonarchHeader_RunType int32

const (
	MonarchHeader_Background MonarchHeader_RunType = 0
	MonarchHeader_Signal     MonarchHeader_RunType = 1
	MonarchHeader_Other      MonarchHeader_RunType = 999
)

var MonarchHeader_RunType_name = map[int32]string{
	0:   "Background",
	1:   "Signal",
	999: "Other",
}
var MonarchHeader_RunType_value = map[string]int32{
	"Background": 0,
	"Signal":     1,
	"Other":      999,
}

func (x MonarchHeader_RunType) Enum() *MonarchHeader_RunType {
	p := new(MonarchHeader_RunType)
	*p = x
	return p
}
func (x MonarchHeader_RunType) String() string {
	return proto.EnumName(MonarchHeader_RunType_name, int32(x))
}
func (x MonarchHeader_RunType) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
func (x *MonarchHeader_RunType) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(MonarchHeader_RunType_value, data, "MonarchHeader_RunType")
	if err != nil {
		return err
	}
	*x = MonarchHeader_RunType(value)
	return nil
}

type MonarchHeader_FormatMode int32

const (
	MonarchHeader_Single           MonarchHeader_FormatMode = 0
	MonarchHeader_MultiSeparate    MonarchHeader_FormatMode = 1
	MonarchHeader_MultiInterleaved MonarchHeader_FormatMode = 2
)

var MonarchHeader_FormatMode_name = map[int32]string{
	0: "Single",
	1: "MultiSeparate",
	2: "MultiInterleaved",
}
var MonarchHeader_FormatMode_value = map[string]int32{
	"Single":           0,
	"MultiSeparate":    1,
	"MultiInterleaved": 2,
}

func (x MonarchHeader_FormatMode) Enum() *MonarchHeader_FormatMode {
	p := new(MonarchHeader_FormatMode)
	*p = x
	return p
}
func (x MonarchHeader_FormatMode) String() string {
	return proto.EnumName(MonarchHeader_FormatMode_name, int32(x))
}
func (x MonarchHeader_FormatMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(x.String())
}
func (x *MonarchHeader_FormatMode) UnmarshalJSON(data []byte) error {
	value, err := proto.UnmarshalJSONEnum(MonarchHeader_FormatMode_value, data, "MonarchHeader_FormatMode")
	if err != nil {
		return err
	}
	*x = MonarchHeader_FormatMode(value)
	return nil
}

type MonarchHeader struct {
	Filename         *string                   `protobuf:"bytes,1,req,name=filename" json:"filename,omitempty"`
	AcqRate          *float64                  `protobuf:"fixed64,2,req,name=acqRate" json:"acqRate,omitempty"`
	AcqMode          *uint32                   `protobuf:"varint,3,req,name=acqMode" json:"acqMode,omitempty"`
	AcqTime          *uint32                   `protobuf:"varint,4,req,name=acqTime" json:"acqTime,omitempty"`
	RecSize          *uint32                   `protobuf:"varint,5,req,name=recSize" json:"recSize,omitempty"`
	RunDate          *string                   `protobuf:"bytes,6,opt,name=runDate,def=(unknown)" json:"runDate,omitempty"`
	RunInfo          *string                   `protobuf:"bytes,7,opt,name=runInfo,def=(unknown)" json:"runInfo,omitempty"`
	RunSource        *MonarchHeader_RunSource  `protobuf:"varint,8,opt,name=runSource,enum=Protobuf.MonarchHeader_RunSource" json:"runSource,omitempty"`
	RunType          *MonarchHeader_RunType    `protobuf:"varint,9,opt,name=runType,enum=Protobuf.MonarchHeader_RunType" json:"runType,omitempty"`
	FormatMode       *MonarchHeader_FormatMode `protobuf:"varint,10,opt,name=formatMode,enum=Protobuf.MonarchHeader_FormatMode,def=2" json:"formatMode,omitempty"`
	XXX_unrecognized []byte                    `json:"-"`
}

func (m *MonarchHeader) Reset()         { *m = MonarchHeader{} }
func (m *MonarchHeader) String() string { return proto.CompactTextString(m) }
func (*MonarchHeader) ProtoMessage()    {}

const Default_MonarchHeader_RunDate string = "(unknown)"
const Default_MonarchHeader_RunInfo string = "(unknown)"
const Default_MonarchHeader_FormatMode MonarchHeader_FormatMode = MonarchHeader_MultiInterleaved

func (m *MonarchHeader) GetFilename() string {
	if m != nil && m.Filename != nil {
		return *m.Filename
	}
	return ""
}

func (m *MonarchHeader) GetAcqRate() float64 {
	if m != nil && m.AcqRate != nil {
		return *m.AcqRate
	}
	return 0
}

func (m *MonarchHeader) GetAcqMode() uint32 {
	if m != nil && m.AcqMode != nil {
		return *m.AcqMode
	}
	return 0
}

func (m *MonarchHeader) GetAcqTime() uint32 {
	if m != nil && m.AcqTime != nil {
		return *m.AcqTime
	}
	return 0
}

func (m *MonarchHeader) GetRecSize() uint32 {
	if m != nil && m.RecSize != nil {
		return *m.RecSize
	}
	return 0
}

func (m *MonarchHeader) GetRunDate() string {
	if m != nil && m.RunDate != nil {
		return *m.RunDate
	}
	return Default_MonarchHeader_RunDate
}

func (m *MonarchHeader) GetRunInfo() string {
	if m != nil && m.RunInfo != nil {
		return *m.RunInfo
	}
	return Default_MonarchHeader_RunInfo
}

func (m *MonarchHeader) GetRunSource() MonarchHeader_RunSource {
	if m != nil && m.RunSource != nil {
		return *m.RunSource
	}
	return 0
}

func (m *MonarchHeader) GetRunType() MonarchHeader_RunType {
	if m != nil && m.RunType != nil {
		return *m.RunType
	}
	return 0
}

func (m *MonarchHeader) GetFormatMode() MonarchHeader_FormatMode {
	if m != nil && m.FormatMode != nil {
		return *m.FormatMode
	}
	return Default_MonarchHeader_FormatMode
}

func init() {
	proto.RegisterEnum("Protobuf.MonarchHeader_RunSource", MonarchHeader_RunSource_name, MonarchHeader_RunSource_value)
	proto.RegisterEnum("Protobuf.MonarchHeader_RunType", MonarchHeader_RunType_name, MonarchHeader_RunType_value)
	proto.RegisterEnum("Protobuf.MonarchHeader_FormatMode", MonarchHeader_FormatMode_name, MonarchHeader_FormatMode_value)
}
