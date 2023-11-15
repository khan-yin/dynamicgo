<!-- Code generated by gomarkdoc. DO NOT EDIT -->

# proto

```go
import "github.com/cloudwego/dynamicgo/proto"
```

## Introduction
**DynamicGo For Protobuf Protocol：**[INTRODUCTION.md](./INTRODUCTION.md)

## Index

- [proto](#proto)
  - [Introduction](#introduction)
  - [Index](#index)
  - [Variables](#variables)
  - [type Descriptor](#type-descriptor)
  - [type EnumDescriptor](#type-enumdescriptor)
  - [type EnumNumber](#type-enumnumber)
  - [type EnumValueDescriptor](#type-enumvaluedescriptor)
  - [type ExtensionDescriptor](#type-extensiondescriptor)
  - [type ExtensionTypeDescriptor](#type-extensiontypedescriptor)
  - [type FieldDescriptor](#type-fielddescriptor)
  - [type FieldDescriptors](#type-fielddescriptors)
  - [type FieldName](#type-fieldname)
  - [type FieldNumber](#type-fieldnumber)
  - [type FileDescriptor](#type-filedescriptor)
  - [type MessageDescriptor](#type-messagedescriptor)
    - [func FnRequest](#func-fnrequest)
  - [type MethodDescriptor](#type-methoddescriptor)
    - [func GetFnDescFromFile](#func-getfndescfromfile)
  - [type Number](#type-number)
  - [type OneofDescriptor](#type-oneofdescriptor)
  - [type Options](#type-options)
    - [func NewDefaultOptions](#func-newdefaultoptions)
    - [func (Options) NewDescriptorFromPath](#func-options-newdescriptorfrompath)
  - [type ParseTarget](#type-parsetarget)
  - [type ProtoKind](#type-protokind)
  - [type ServiceDescriptor](#type-servicedescriptor)
    - [func NewDescritorFromPath](#func-newdescritorfrompath)
  - [type Type](#type-type)
    - [func FromProtoKindToType](#func-fromprotokindtotype)
    - [func (Type) IsComplex](#func-type-iscomplex)
    - [func (Type) IsInt](#func-type-isint)
    - [func (Type) IsUint](#func-type-isuint)
    - [func (Type) NeedVarint](#func-type-needvarint)
    - [func (Type) String](#func-type-string)
    - [func (Type) TypeToKind](#func-type-typetokind)
    - [func (Type) Valid](#func-type-valid)
  - [type WireType](#type-wiretype)
    - [func (WireType) String](#func-wiretype-string)


## Variables

<a name="Kind2Wire"></a>map from proto.ProtoKind to proto.WireType

```go
var Kind2Wire = map[ProtoKind]WireType{
    BoolKind:     VarintType,
    EnumKind:     VarintType,
    Int32Kind:    VarintType,
    Sint32Kind:   VarintType,
    Uint32Kind:   VarintType,
    Int64Kind:    VarintType,
    Sint64Kind:   VarintType,
    Uint64Kind:   VarintType,
    Sfixed32Kind: Fixed32Type,
    Fixed32Kind:  Fixed32Type,
    FloatKind:    Fixed32Type,
    Sfixed64Kind: Fixed64Type,
    Fixed64Kind:  Fixed64Type,
    DoubleKind:   Fixed64Type,
    StringKind:   BytesType,
    BytesKind:    BytesType,
    MessageKind:  BytesType,
    GroupKind:    StartGroupType,
}
```

<a name="Descriptor"></a>
## type [Descriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L7>)



```go
type Descriptor = protoreflect.Descriptor
```

<a name="EnumDescriptor"></a>
## type [EnumDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L20>)

special fielddescriptor for enum, oneof, extension, extensionrange

```go
type EnumDescriptor = protoreflect.EnumDescriptor
```

<a name="EnumNumber"></a>
## type [EnumNumber](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L217>)



```go
type EnumNumber = Number
```

<a name="EnumValueDescriptor"></a>
## type [EnumValueDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L22>)



```go
type EnumValueDescriptor = protoreflect.EnumValueDescriptor
```

<a name="ExtensionDescriptor"></a>
## type [ExtensionDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L27>)

ExtensionDescriptor is the same as FieldDescriptor

```go
type ExtensionDescriptor = protoreflect.ExtensionDescriptor
```

<a name="ExtensionTypeDescriptor"></a>
## type [ExtensionTypeDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L29>)



```go
type ExtensionTypeDescriptor = protoreflect.ExtensionTypeDescriptor
```

<a name="FieldDescriptor"></a>
## type [FieldDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L17>)



```go
type FieldDescriptor = protoreflect.FieldDescriptor
```

<a name="FieldDescriptors"></a>
## type [FieldDescriptors](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L31>)



```go
type FieldDescriptors = protoreflect.FieldDescriptors
```

<a name="FieldName"></a>
## type [FieldName](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L229>)

define FieldName = protoreflect.Name (string) used in Descriptor.Name()

```go
type FieldName = protoreflect.Name
```

<a name="FieldNumber"></a>
## type [FieldNumber](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L216>)



```go
type FieldNumber = Number
```

<a name="FileDescriptor"></a>
## type [FileDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L9>)



```go
type FileDescriptor = protoreflect.FieldDescriptor
```

<a name="MessageDescriptor"></a>
## type [MessageDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L15>)



```go
type MessageDescriptor = protoreflect.MessageDescriptor
```

<a name="FnRequest"></a>
### func [FnRequest](<https://github.com/khan-yin/dynamicgo/blob/main/proto/test_util.go#L26>)

```go
func FnRequest(fn *MethodDescriptor) *MessageDescriptor
```

FnRequest get the normal request type

<a name="MethodDescriptor"></a>
## type [MethodDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L13>)



```go
type MethodDescriptor = protoreflect.MethodDescriptor
```

<a name="GetFnDescFromFile"></a>
### func [GetFnDescFromFile](<https://github.com/khan-yin/dynamicgo/blob/main/proto/test_util.go#L13>)

```go
func GetFnDescFromFile(filePath, fnName string, opts Options) *MethodDescriptor
```

GetFnDescFromFile get a fucntion descriptor from idl path (relative to your git root) and the function name

<a name="Number"></a>
## type [Number](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L214>)

define Number = protowire.Number (int32)

```go
type Number = protowire.Number
```

<a name="MinValidNumber"></a>reserved field number min-max ranges in a proto message

```go
const (
    MinValidNumber        Number = 1
    FirstReservedNumber   Number = 19000
    LastReservedNumber    Number = 19999
    MaxValidNumber        Number = 1<<29 - 1
    DefaultRecursionLimit        = 10000
)
```

<a name="OneofDescriptor"></a>
## type [OneofDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L24>)



```go
type OneofDescriptor = protoreflect.OneofDescriptor
```

<a name="Options"></a>
## type [Options](<https://github.com/khan-yin/dynamicgo/blob/main/proto/idl.go#L23-L40>)

Options is options for parsing thrift IDL.

```go
type Options struct {
    // ParseServiceMode indicates how to parse service.
    ParseServiceMode meta.ParseServiceMode

    MapFieldWay meta.MapFieldWay // not implemented.

    ParseFieldRandomRate float64 // not implemented.

    ParseEnumAsInt64 bool // not implemented.

    SetOptionalBitmap bool // not implemented.

    UseDefaultValue bool // not implemented.

    ParseFunctionMode meta.ParseFunctionMode // not implemented.

    EnableProtoBase bool // not implemented.
}
```

<a name="NewDefaultOptions"></a>
### func [NewDefaultOptions](<https://github.com/khan-yin/dynamicgo/blob/main/proto/idl.go#L43>)

```go
func NewDefaultOptions() Options
```

NewDefaultOptions creates a default Options.

<a name="Options.NewDescriptorFromPath"></a>
### func (Options) [NewDescriptorFromPath](<https://github.com/khan-yin/dynamicgo/blob/main/proto/idl.go#L54>)

```go
func (opts Options) NewDescriptorFromPath(ctx context.Context, path string, importDirs ...string) (*ServiceDescriptor, error)
```

NewDescritorFromContent creates a ServiceDescriptor from a proto path and its imports, which uses the given options. The importDirs is used to find the include files.

<a name="ParseTarget"></a>
## type [ParseTarget](<https://github.com/khan-yin/dynamicgo/blob/main/proto/idl.go#L20>)

ParseTarget indicates the target to parse

```go
type ParseTarget uint8
```

<a name="Request"></a>

```go
const (
    Request ParseTarget = iota
    Response
    Exception
)
```

<a name="ProtoKind"></a>
## type [ProtoKind](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L40>)

define ProtoKind = protoreflect.Kind (int8)

```go
type ProtoKind = protoreflect.Kind
```

<a name="DoubleKind"></a>

```go
const (
    DoubleKind ProtoKind = iota + 1
    FloatKind
    Int64Kind
    Uint64Kind
    Int32Kind
    Fixed64Kind
    Fixed32Kind
    BoolKind
    StringKind
    GroupKind
    MessageKind
    BytesKind
    Uint32Kind
    EnumKind
    Sfixed32Kind
    Sfixed64Kind
    Sint32Kind
    Sint64Kind
)
```

<a name="ServiceDescriptor"></a>
## type [ServiceDescriptor](<https://github.com/khan-yin/dynamicgo/blob/main/proto/descriptor.go#L11>)



```go
type ServiceDescriptor = protoreflect.ServiceDescriptor
```

<a name="NewDescritorFromPath"></a>
### func [NewDescritorFromPath](<https://github.com/khan-yin/dynamicgo/blob/main/proto/idl.go#L48>)

```go
func NewDescritorFromPath(ctx context.Context, path string, importDirs ...string) (*ServiceDescriptor, error)
```

NewDescritorFromPath behaviors like NewDescritorFromPath, besides it uses DefaultOptions.

<a name="Type"></a>
## type [Type](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L86>)

Node type (uint8) mapping ProtoKind the same value, except for UNKNOWN, LIST, MAP, ERROR

```go
type Type uint8
```

<a name="UNKNOWN"></a>

```go
const (
    UNKNOWN Type = 0 // unknown field type
    DOUBLE  Type = 1
    FLOAT   Type = 2
    INT64   Type = 3
    UINT64  Type = 4
    INT32   Type = 5
    FIX64   Type = 6
    FIX32   Type = 7
    BOOL    Type = 8
    STRING  Type = 9
    GROUP   Type = 10 // deprecated
    MESSAGE Type = 11
    BYTE    Type = 12
    UINT32  Type = 13
    ENUM    Type = 14
    SFIX32  Type = 15
    SFIX64  Type = 16
    SINT32  Type = 17
    SINT64  Type = 18
    LIST    Type = 19
    MAP     Type = 20
    ERROR   Type = 255
)
```

<a name="FromProtoKindToType"></a>
### func [FromProtoKindToType](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L136>)

```go
func FromProtoKindToType(kind ProtoKind, isList bool, isMap bool) Type
```

FromProtoKindTType converts ProtoKind to Type

<a name="Type.IsComplex"></a>
### func (Type) [IsComplex](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L161>)

```go
func (p Type) IsComplex() bool
```

IsComplex tells if the type is one of STRUCT, MAP, SET, LIST

<a name="Type.IsInt"></a>
### func (Type) [IsInt](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L152>)

```go
func (p Type) IsInt() bool
```

IsInt containing isUint

<a name="Type.IsUint"></a>
### func (Type) [IsUint](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L156>)

```go
func (p Type) IsUint() bool
```



<a name="Type.NeedVarint"></a>
### func (Type) [NeedVarint](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L147>)

```go
func (p Type) NeedVarint() bool
```

check if the type need Varint encoding, also used in check list isPacked

<a name="Type.String"></a>
### func (Type) [String](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L166>)

```go
func (p Type) String() string
```

String for format and print

<a name="Type.TypeToKind"></a>
### func (Type) [TypeToKind](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L123>)

```go
func (p Type) TypeToKind() ProtoKind
```



<a name="Type.Valid"></a>
### func (Type) [Valid](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L113>)

```go
func (p Type) Valid() bool
```



<a name="WireType"></a>
## type [WireType](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L9>)

protobuf encoding wire type

```go
type WireType int8
```

<a name="VarintType"></a>

```go
const (
    VarintType     WireType = 0
    Fixed32Type    WireType = 5
    Fixed64Type    WireType = 1
    BytesType      WireType = 2
    StartGroupType WireType = 3 // deprecated
    EndGroupType   WireType = 4 // deprecated
)
```

<a name="WireType.String"></a>
### func (WireType) [String](<https://github.com/khan-yin/dynamicgo/blob/main/proto/type.go#L20>)

```go
func (p WireType) String() string
```



Generated by [gomarkdoc](<https://github.com/princjef/gomarkdoc>)