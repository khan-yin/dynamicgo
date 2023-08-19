package base

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"sync"
	"unicode/utf8"
	"unsafe"

	"github.com/cloudwego/dynamicgo/internal/primitive"
	"github.com/cloudwego/dynamicgo/internal/rt"
	"github.com/cloudwego/dynamicgo/meta"
	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/dynamicgo/proto/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// memory resize factor
const (
	defaultBufferSize = 4096
	growBufferFactor  = 1
	defaultListSize   = 10
)

// map from proto.ProtoKind to proto.WireType
var wireTypes = map[proto.ProtoKind]proto.WireType{
	proto.BoolKind:     proto.VarintType,
	proto.EnumKind:     proto.VarintType,
	proto.Int32Kind:    proto.VarintType,
	proto.Sint32Kind:   proto.VarintType,
	proto.Uint32Kind:   proto.VarintType,
	proto.Int64Kind:    proto.VarintType,
	proto.Sint64Kind:   proto.VarintType,
	proto.Uint64Kind:   proto.VarintType,
	proto.Sfixed32Kind: proto.Fixed32Type,
	proto.Fixed32Kind:  proto.Fixed32Type,
	proto.FloatKind:    proto.Fixed32Type,
	proto.Sfixed64Kind: proto.Fixed64Type,
	proto.Fixed64Kind:  proto.Fixed64Type,
	proto.DoubleKind:   proto.Fixed64Type,
	proto.StringKind:   proto.BytesType,
	proto.BytesKind:    proto.BytesType,
	proto.MessageKind:  proto.BytesType,
	proto.GroupKind:    proto.StartGroupType,
}

var (
	errDismatchPrimitive = meta.NewError(meta.ErrDismatchType, "dismatch primitive types", nil)
	errInvalidDataSize   = meta.NewError(meta.ErrInvalidParam, "invalid data size", nil)
	errInvalidTag        = meta.NewError(meta.ErrInvalidParam, "invalid tag in ReadMessageBegin", nil)
	errExceedDepthLimit  = meta.NewError(meta.ErrStackOverflow, "exceed depth limit", nil)
	errInvalidDataType   = meta.NewError(meta.ErrRead, "invalid data type", nil)
	errUnknonwField      = meta.NewError(meta.ErrUnknownField, "unknown field", nil)
	errUnsupportedType   = meta.NewError(meta.ErrUnsupportedType, "unsupported type", nil)
	errNotImplemented    = meta.NewError(meta.ErrNotImplemented, "not implemted type", nil)
	errCodeFieldNumber   = meta.NewError(meta.ErrConvert, "invalid field number", nil)
	errDecodeField       = meta.NewError(meta.ErrRead, "cannot parse invalid wire-format data", nil)
)

// We append to an empty array rather than a nil []byte to get non-nil zero-length byte slices.
var emptyBuf [0]byte

// Serizalize data to byte array and reuse the memory
type BinaryProtocol struct {
	Buf  []byte
	Read int
}

var (
	bpPool = sync.Pool{
		New: func() interface{} {
			return &BinaryProtocol{
				Buf: make([]byte, 0, defaultBufferSize),
			}
		},
	}
)

func (p *BinaryProtocol) malloc(size int) ([]byte, error) {
	if size <= 0 {
		panic(errors.New("invalid size"))
	}

	l := len(p.Buf)
	c := cap(p.Buf)
	d := l + size

	if d > c {
		c += c >> growBufferFactor
		if d > c {
			c = d * 2
		}
		buf := rt.Growslice(byteType, *(*rt.GoSlice)(unsafe.Pointer(&p.Buf)), c)
		p.Buf = *(*[]byte)(unsafe.Pointer(&buf))
	}
	p.Buf = (p.Buf)[:d]

	return (p.Buf)[l:d], nil
}

func NewBinaryProtol(buf []byte) *BinaryProtocol {
	bp := bpPool.Get().(*BinaryProtocol)
	bp.Buf = buf
	return bp
}

func NewBinaryProtocolBuffer() *BinaryProtocol {
	bp := bpPool.Get().(*BinaryProtocol)
	return bp
}

func FreeBinaryProtocol(bp *BinaryProtocol) {
	bp.Reset()
	bpPool.Put(bp)
}

func (p *BinaryProtocol) Recycle() {
	p.Reset()
	bpPool.Put(p)
}

// Reset resets the buffer and read position
func (p *BinaryProtocol) Reset() {
	p.Read = 0
	p.Buf = p.Buf[:0]
}

// RawBuf returns the raw buffer of the protocol
func (p *BinaryProtocol) RawBuf() []byte {
	return p.Buf
}

// Left returns the left bytes to read
func (p *BinaryProtocol) Left() int {
	return len(p.Buf) - p.Read
}

// Append Tag
func (p *BinaryProtocol) AppendTag(num proto.Number, typ proto.WireType) error {
	tag := uint64(num)<<3 | uint64(typ&7)
	p.Buf = protowire.BinaryEncoder{}.EncodeInt64(p.Buf, int64(tag))
	return nil
}

// ConsumeTag parses b as a varint-encoded tag, reporting its length.
func (p *BinaryProtocol) ConsumeTag() (proto.Number, proto.WireType, int, error) {
	v, n := protowire.ConsumeVarint((p.Buf)[p.Read:])
	_, err := p.next(n)
	if n < 0 {
		return 0, 0, n, errInvalidTag
	}
	if v>>3 > uint64(math.MaxInt32) {
		return -1, 0, n, errUnknonwField
	}
	num, typ := proto.Number(v>>3), proto.WireType(v&7)
	if num < proto.MinValidNumber {
		return 0, 0, n, errCodeFieldNumber
	}
	return num, typ, n, err
}

// WriteBool
func (p *BinaryProtocol) WriteBool(value bool) error {
	if value {
		return p.WriteUint64(uint64(1))
	} else {
		return p.WriteUint64(uint64(0))
	}
}

// WriteInt32
func (p *BinaryProtocol) WriteI32(value int32) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeInt32(p.Buf, value)
	return nil
}

// WriteSint32
func (p *BinaryProtocol) WriteSint32(value int32) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeSint32(p.Buf, value)
	return nil
}

// WriteUint32
func (p *BinaryProtocol) WriteUint32(value uint32) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeUint32(p.Buf, value)
	return nil
}

// Writefixed32
func (p *BinaryProtocol) WriteFixed32(value int32) error {
	v, err := p.malloc(4)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(v, uint32(value))
	return err
}

// WriteSfixed32
func (p *BinaryProtocol) WriteSfixed32(value int32) error {
	v, err := p.malloc(4)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(v, uint32(value))
	return err
}

// WriteInt64
func (p *BinaryProtocol) WriteI64(value int64) error {
	protowire.BinaryEncoder{}.EncodeInt64(p.Buf, value)
	return nil
}

// WriteSint64
func (p *BinaryProtocol) WriteSint64(value int64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeSint64(p.Buf, value)
	return nil
}

// WriteUint64
func (p *BinaryProtocol) WriteUint64(value uint64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeUint64(p.Buf, value)
	return nil
}

// Writefixed64
func (p *BinaryProtocol) WriteFixed64(value uint64) error {
	v, err := p.malloc(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(v, value)
	return err
}

// WriteSfixed64
func (p *BinaryProtocol) WriteSfixed64(value int64) error {
	v, err := p.malloc(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(v, uint64(value))
	return err
}

// WriteFloat
func (p *BinaryProtocol) WriteFloat(value float64) error {
	v, err := p.malloc(4)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(v, math.Float32bits(float32(value)))
	return err
}

// WriteDouble
func (p *BinaryProtocol) WriteDouble(value float64) error {
	v, err := p.malloc(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(v, math.Float64bits(value))
	return err
}

// WriteString
func (p *BinaryProtocol) WriteString(value string) error {
	if !utf8.ValidString(value) {
		return meta.NewError(meta.ErrInvalidParam, value, nil)
	}
	p.Buf = protowire.BinaryEncoder{}.EncodeString(p.Buf, value)
	return nil
}

// WriteBytes
func (p *BinaryProtocol) WriteBytes(value []byte) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeBytes(p.Buf, value)
	return nil
}

// WriteEnum
func (p *BinaryProtocol) WriteEnum(value proto.EnumNumber) error {
	protowire.BinaryEncoder{}.EncodeInt64(p.Buf, int64(value))
	return nil
}

/**
 * WriteList
 */
func (p *BinaryProtocol) WriteList(desc *proto.FieldDescriptor, val interface{}) error {
	vs, ok := val.([]interface{})
	if !ok {
		return errDismatchPrimitive
	}
	// packed List bytes format: [tag][length][T(L)V][value][value]...
	fd := *desc
	if fd.IsPacked() && len(vs) > 0 {
		p.AppendTag(fd.Number(), proto.BytesType)
		var pos int
		p.Buf, pos = appendSpeculativeLength(p.Buf)
		for _, v := range vs {

			if err := p.WriteAnyWithDesc(desc, v, true, false, true); err != nil {
				return err
			}
		}
		p.Buf = finishSpeculativeLength(p.Buf, pos)
		return nil
	}

	// unpacked List bytes format: [T(L)V][T(L)V]...
	kind := fd.Kind()
	for _, v := range vs {
		// share the same field number for Tag
		p.AppendTag(fd.Number(), wireTypes[kind])
		if err := p.WriteAnyWithDesc(desc, v, true, false, true); err != nil {
			return err
		}
	}
	return nil
}

/**
 * WriteMap
 */
func (p *BinaryProtocol) WriteMap(desc *proto.FieldDescriptor, val interface{}) error {
	fd := *desc
	MapKey := fd.MapKey()
	MapValue := fd.MapValue()
	vs, ok := val.(map[interface{}]interface{})
	if !ok {
		return errDismatchPrimitive
	}

	for k, v := range vs {
		p.AppendTag(fd.Number(), proto.BytesType)
		var pos int
		p.Buf, pos = appendSpeculativeLength(p.Buf)
		p.WriteAnyWithDesc(&MapKey, k, true, false, true)
		p.WriteAnyWithDesc(&MapValue, v, true, false, true)
		p.Buf = finishSpeculativeLength(p.Buf, pos)
	}

	return nil
}

/**
 * Write Message
 */
func (p *BinaryProtocol) WriteMessageSlow(desc *proto.FieldDescriptor, vs map[string]interface{}, cast bool, disallowUnknown bool, useFieldName bool) error {
	for id, v := range vs {
		f := (*desc).Message().Fields().ByName(protoreflect.Name(id))
		if f == nil {
			if disallowUnknown {
				return errUnknonwField
			}
			continue
		}
		if e := p.WriteAnyWithDesc(&f, v, cast, disallowUnknown, useFieldName); e != nil {
			return e
		}
	}
	return nil
}

// When encoding length-prefixed fields, we speculatively set aside some number of bytes
// for the length, encode the data, and then encode the length (shifting the data if necessary
// to make room).
const speculativeLength = 1

func appendSpeculativeLength(b []byte) ([]byte, int) {
	pos := len(b)
	b = append(b, "\x00\x00\x00\x00"[:speculativeLength]...)
	return b, pos
}

func finishSpeculativeLength(b []byte, pos int) []byte {
	mlen := len(b) - pos - speculativeLength
	msiz := protowire.SizeVarint(uint64(mlen))
	if msiz != speculativeLength {
		for i := 0; i < msiz-speculativeLength; i++ {
			b = append(b, 0)
		}
		copy(b[pos+msiz:], b[pos+speculativeLength:])
		b = b[:pos+msiz+mlen]
	}
	protowire.AppendVarint(b[:pos], uint64(mlen))
	return b
}

// WriteAnyWithDesc explain desc and val and write them into buffer
//   - LIST/SET will be converted from []interface{}
//   - MAP will be converted from map[string]interface{} or map[int]interface{}
//   - STRUCT will be converted from map[FieldID]interface{}
func (p *BinaryProtocol) WriteAnyWithDesc(desc *proto.FieldDescriptor, val interface{}, cast bool, disallowUnknown bool, useFieldName bool) error {
	fd := *desc
	switch {
	case fd.IsList():
		return p.WriteList(desc, val)
	case fd.IsMap():
		return p.WriteMap(desc, val)
	default:
		if e := p.AppendTag(proto.Number(fd.Number()), wireTypes[fd.Kind()]); e != nil {
			return meta.NewError(meta.ErrWrite, "AppenddescTag failed", nil)
		}
		return p.WriteBaseTypeWithDesc(desc, val, cast, disallowUnknown, useFieldName)
	}
}

// WriteBaseType with desc, not thread safe
func (p *BinaryProtocol) WriteBaseTypeWithDesc(fd *proto.FieldDescriptor, val interface{}, cast bool, disallowUnknown bool, useFieldName bool) error {
	switch (*fd).Kind() {
	case protoreflect.Kind(proto.BoolKind):
		v, ok := val.(bool)
		if !ok {
			var err error
			v, err = primitive.ToBool(val)
			if err != nil {
				return err
			}
		}
		p.WriteBool(v)
	case protoreflect.Kind(proto.EnumKind):
		v, ok := val.(proto.EnumNumber)
		if !ok {
			return meta.NewError(meta.ErrConvert, "WriteEnum error", nil)
		}
		p.WriteEnum(v)
	case protoreflect.Kind(proto.Int32Kind):
		v, ok := val.(int32)
		if !ok {
			var err error
			vv, err := primitive.ToInt64(v)
			if err != nil {
				return err
			}
			v = int32(vv)
		}
		p.WriteI32(v)
	case protoreflect.Kind(proto.Sint32Kind):
		v, ok := val.(int32)
		if !ok {
			var err error
			vv, err := primitive.ToInt64(v)
			if err != nil {
				return err
			}
			v = int32(vv)
		}
		p.WriteSint32(v)
	case protoreflect.Kind(proto.Uint32Kind):
		v, ok := val.(uint32)
		if !ok {
			var err error
			vv, err := primitive.ToInt64(v)
			if err != nil {
				return err
			}
			v = uint32(vv)
		}
		p.WriteUint32(v)
	case protoreflect.Kind(proto.Int64Kind):
		v, ok := val.(int64)
		if !ok {
			var err error
			v, err = primitive.ToInt64(v)
			if err != nil {
				return err
			}
		}
		p.WriteI64(v)
	case protoreflect.Kind(proto.Sint64Kind):
		v, ok := val.(int64)
		if !ok {
			var err error
			v, err = primitive.ToInt64(v)
			if err != nil {
				return err
			}
		}
		p.WriteSint64(v)
	case protoreflect.Kind(proto.Uint64Kind):
		v, ok := val.(uint64)
		if !ok {
			var err error
			vv, err := primitive.ToInt64(v)
			if err != nil {
				return err
			}
			v = uint64(vv)
		}
		p.WriteUint64(v)
	case protoreflect.Kind(proto.Sfixed32Kind):
		v, ok := val.(int32)
		if !ok {
			var err error
			vv, err := primitive.ToInt64(v)
			if err != nil {
				return err
			}
			v = int32(vv)
		}
		p.WriteSfixed32(v)
	case protoreflect.Kind(proto.Fixed32Kind):
		v, ok := val.(int32)
		if !ok {
			var err error
			vv, err := primitive.ToInt64(v)
			if err != nil {
				return err
			}
			v = int32(vv)
		}
		p.WriteFixed32(v)
	case protoreflect.Kind(proto.FloatKind):
		v, ok := val.(float64)
		if !ok {
			var err error
			v, err = primitive.ToFloat64(v)
			if err != nil {
				return err
			}
		}
		p.WriteFloat(v)
	case protoreflect.Kind(proto.Sfixed64Kind):
		v, ok := val.(int64)
		if !ok {
			var err error
			v, err = primitive.ToInt64(v)
			if err != nil {
				return err
			}
		}
		p.WriteSfixed64(v)
	case protoreflect.Kind(proto.Fixed64Kind):
		v, ok := val.(int64)
		if !ok {
			var err error
			v, err = primitive.ToInt64(v)
			if err != nil {
				return err
			}
		}
		p.WriteSfixed64(v)
	case protoreflect.Kind(proto.DoubleKind):
		v, ok := val.(float64)
		if !ok {
			var err error
			v, err = primitive.ToFloat64(v)
			if err != nil {
				return err
			}
		}
		p.WriteDouble(v)
	case protoreflect.Kind(proto.StringKind):
		v, ok := val.(string)
		if !ok {
			return meta.NewError(meta.ErrConvert, string((*fd).FullName()), nil)
		}
		p.WriteString(v)
	case protoreflect.Kind(proto.BytesKind):
		v, ok := val.([]byte)
		if !ok {
			return errors.New("WriteBytesType error")
		}
		p.WriteBytes(v)
	case protoreflect.Kind(proto.MessageKind):
		var pos int
		var err error
		vs, ok := val.(map[string]interface{})
		if !ok {
			return errDismatchPrimitive
		}
		p.Buf, pos = appendSpeculativeLength(p.Buf)
		err = p.WriteMessageSlow(fd, vs, cast, disallowUnknown, useFieldName)
		if err != nil {
			return err
		}
		p.Buf = finishSpeculativeLength(p.Buf, pos)
	// case protoreflect.Kind(proto.GroupKind):
	// 	var err error
	// 	b, err = o.marshalMessage(b, v.Message())
	// 	if err != nil {
	// 		return b, err
	// 	}
	// 	b = protowire.AppendVarint(b, protowire.EncodeTag(fd.Number(), protowire.EndGroupType))
	default:
		return errUnsupportedType
	}
	return nil
}

// next ...
func (p *BinaryProtocol) next(size int) ([]byte, error) {
	if size <= 0 {
		panic(errors.New("invalid size"))
	}

	l := len(p.Buf)
	d := p.Read + size
	if d > l {
		return nil, io.EOF
	}

	ret := (p.Buf)[p.Read:d]
	p.Read = d
	return ret, nil
}

// ReadByte
func (p *BinaryProtocol) ReadByte() (value byte, err error) {
	buf, err := p.next(1)
	if err != nil {
		return value, err
	}
	return byte(buf[0]), err
}

// ReadBool
func (p *BinaryProtocol) ReadBool() (bool, error) {
	v, n := protowire.BinaryDecoder{}.DecodeBool((p.Buf)[p.Read:])
	if n < 0 {
		return false, errDecodeField
	}
	_, err := p.next(n)
	return v, err
}

// ReadI32
func (p *BinaryProtocol) ReadI32() (int32, error) {
	value, n := protowire.BinaryDecoder{}.DecodeInt32((p.Buf)[p.Read:])
	if n < 0 {
		return value, errDecodeField
	}
	_, err := p.next(n)
	return value, err
}

// ReadSint32
func (p *BinaryProtocol) ReadSint32() (int32, error) {
	value, n := protowire.BinaryDecoder{}.DecodeSint32((p.Buf)[p.Read:])
	if n < 0 {
		return value, errDecodeField
	}
	_, err := p.next(n)
	return value, err
}

// ReadUint32
func (p *BinaryProtocol) ReadUint32() (uint32, error) {
	value, n := protowire.BinaryDecoder{}.DecodeUint32((p.Buf)[p.Read:])
	if n < 0 {
		return value, errDecodeField
	}
	_, err := p.next(n)
	return value, err
}

// ReadI64
func (p *BinaryProtocol) ReadI64() (int64, error) {
	value, n := protowire.BinaryDecoder{}.DecodeInt64((p.Buf)[p.Read:])
	if n < 0 {
		return value, errDecodeField
	}
	_, err := p.next(n)
	return value, err
}

// ReadSint64
func (p *BinaryProtocol) ReadSint64() (int64, error) {
	value, n := protowire.BinaryDecoder{}.DecodeSint64((p.Buf)[p.Read:])
	if n < 0 {
		return value, errDecodeField
	}
	_, err := p.next(n)
	return value, err
}

// ReadUint64
func (p *BinaryProtocol) ReadUint64() (uint64, error) {
	value, n := protowire.BinaryDecoder{}.DecodeUint64((p.Buf)[p.Read:])
	if n < 0 {
		return value, errDecodeField
	}
	_, err := p.next(n)
	return value, err
}

// ReadVarint return data、length、error
func (p *BinaryProtocol) ReadVarint() (uint64, int, error) {
	value, n := protowire.BinaryDecoder{}.DecodeUint64((p.Buf)[p.Read:])
	if n < 0 {
		return value, -1, errDecodeField
	}
	_, err := p.next(n)
	return value, n, err
}

// ReadFixed32
func (p *BinaryProtocol) ReadFixed32() (int32, error) {
	value, n := protowire.BinaryDecoder{}.DecodeFixed32((p.Buf)[p.Read:])
	if n < 0 {
		return int32(value), errDecodeField
	}
	_, err := p.next(n)
	return int32(value), err
}

// ReadSFixed32
func (p *BinaryProtocol) ReadSfixed32() (int32, error) {
	value, n := protowire.BinaryDecoder{}.DecodeFixed32((p.Buf)[p.Read:])
	if n < 0 {
		return int32(value), errDecodeField
	}
	_, err := p.next(n)
	return int32(value), err
}

// ReadFloat
func (p *BinaryProtocol) ReadFloat() (float32, error) {
	value, n := protowire.BinaryDecoder{}.DecodeFloat32((p.Buf)[p.Read:])
	if n < 0 {
		return value, errDecodeField
	}
	_, err := p.next(n)
	return value, err
}

// ReadFixed64
func (p *BinaryProtocol) ReadFixed64() (int64, error) {
	value, n := protowire.BinaryDecoder{}.DecodeFixed64((p.Buf)[p.Read:])
	if n < 0 {
		return int64(value), errDecodeField
	}
	_, err := p.next(n)
	return int64(value), err
}

// ReadSFixed64
func (p *BinaryProtocol) ReadSfixed64() (int64, error) {
	value, n := protowire.BinaryDecoder{}.DecodeFixed64((p.Buf)[p.Read:])
	if n < 0 {
		return int64(value), errDecodeField
	}
	_, err := p.next(n)
	return int64(value), err
}

// ReadDouble
func (p *BinaryProtocol) ReadDouble() (float64, error) {
	value, n := protowire.BinaryDecoder{}.DecodeFixed64((p.Buf)[p.Read:])
	if n < 0 {
		return math.Float64frombits(value), errDecodeField
	}
	_, err := p.next(n)
	return math.Float64frombits(value), err
}

// ReadBytes return bytesData and the sum length of L、V in TLV
func (p *BinaryProtocol) ReadBytes() ([]byte, error) {
	value, n := protowire.BinaryDecoder{}.DecodeBytes((p.Buf)[p.Read:])
	if n < 0 {
		return append(emptyBuf[:], value...), errDecodeField
	}
	_, err := p.next(n)
	return append(emptyBuf[:], value...), err
}

// ReadLength return dataLength, and move pointer in the begin of data
func (p *BinaryProtocol) ReadLength() (int, error) {
	value, n := protowire.BinaryDecoder{}.DecodeUint64((p.Buf)[p.Read:])
	if n < 0 {
		return 0, errDecodeField
	}
	_, err := p.next(n)
	return int(value), err
}

// ReadString
func (p *BinaryProtocol) ReadString(copy bool) (value string, err error) {
	bytes, n := protowire.BinaryDecoder{}.DecodeBytes((p.Buf)[p.Read:])
	if n < 0 {
		return "", errDecodeField
	}
	if copy {
		value = string(bytes)
	} else {
		v := (*rt.GoString)(unsafe.Pointer(&value))
		v.Ptr = rt.IndexPtr(*(*unsafe.Pointer)(unsafe.Pointer(&p.Buf)), byteTypeSize, p.Read)
		v.Len = int(n)
	}

	return
}

// ReadEnum
func (p *BinaryProtocol) ReadEnum() (proto.EnumNumber, error) {
	value, n := protowire.BinaryDecoder{}.DecodeUint64((p.Buf)[p.Read:])
	if n < 0 {
		return 0, errDecodeField
	}
	_, err := p.next(n)
	return proto.EnumNumber(value), err
}

// ReadList
func (p *BinaryProtocol) ReadList() (interface{}, error) {
	return nil, nil
}

// ReadMap
func (p *BinaryProtocol) ReadMap() (interface{}, error) {
	return nil, nil
}

// ReadAnyWithDesc read any type by desc and val
//   - LIST/SET will be converted from []interface{}
//   - MAP will be converted from map[string]interface{} or map[int]interface{}
//   - STRUCT will be converted from map[FieldID]interface{}
func (p *BinaryProtocol) ReadAnyWithDesc(desc *proto.FieldDescriptor, copyString bool, disallowUnknonw bool, useFieldName bool) (interface{}, error) {
	switch {
	case (*desc).IsList():
		return p.ReadList()
	case (*desc).IsMap():
		return p.ReadMap()
	default:
		_, wtyp, _, err := p.ConsumeTag()
		if err != nil {
			return nil, meta.NewError(meta.ErrRead, "ConsumeTag failed", nil)
		}
		return p.ReadBaseTypeWithDesc(desc, wtyp, copyString, disallowUnknonw, useFieldName)
	}
}

// ReadBaseType with desc, not thread safe
func (p *BinaryProtocol) ReadBaseTypeWithDesc(fd *proto.FieldDescriptor, wtyp proto.WireType, copyString bool, disallowUnknown bool, useFieldName bool) (interface{}, error) {
	switch (*fd).Kind() {
	case protoreflect.BoolKind:
		v, e := p.ReadBool()
		return v, e
	case protoreflect.EnumKind:
		v, e := p.ReadEnum()
		return v, e
	case protoreflect.Int32Kind:
		v, e := p.ReadI32()
		return v, e
	case protoreflect.Sint32Kind:
		v, e := p.ReadSint32()
		return v, e
	case protoreflect.Uint32Kind:
		v, e := p.ReadUint32()
		return v, e
	case protoreflect.Fixed32Kind:
		v, e := p.ReadFixed32()
		return v, e
	case protoreflect.Sfixed32Kind:
		v, e := p.ReadSfixed32()
		return v, e
	case protoreflect.Int64Kind:
		v, e := p.ReadI64()
		return v, e
	case protoreflect.Sint64Kind:
		v, e := p.ReadI64()
		return v, e
	case protoreflect.Uint64Kind:
		v, e := p.ReadUint64()
		return v, e
	case protoreflect.Sfixed64Kind:
		v, e := p.ReadSfixed64()
		return v, e
	case protoreflect.FloatKind:
		v, e := p.ReadFloat()
		return v, e
	case protoreflect.DoubleKind:
		v, e := p.ReadDouble()
		return v, e
	case protoreflect.StringKind:
		v, e := p.ReadString(false)
		return v, e
	case protoreflect.BytesKind:
		v, e := p.ReadBytes()
		return v, e
	case protoreflect.MessageKind:
		// get the message data length
		// l, e := p.ReadLength()
		// if e != nil {
		// 	return nil, meta.ErrRead
		// }
		// fields := (*fd).Message().Fields()
		// comma := false
		// existExceptionField := false
		// start := p.Read

		// // *out = json.EncodeObjectBegin(*out)

		// // for p.Read < start+l {
		// // 	fieldId, typeId, _, e := p.ConsumeTag()
		// // 	if e != nil {
		// // 		return wrapError(meta.ErrRead, "", e)
		// // 	}

		// // 	fd := fields.ByNumber(protowire.Number(fieldId))
		// // 	if fd == nil {
		// // 		return wrapError(meta.ErrRead, "invalid field", nil)
		// // 	}

		// // 	if comma {
		// // 		*out = json.EncodeObjectComma(*out)
		// // 	} else {
		// // 		comma = true
		// // 	}

		// // 	// serizalize jsonname
		// // 	*out = json.EncodeString(*out, fd.JSONName())
		// // 	*out = json.EncodeObjectColon(*out)

		// // 	if self.opts.EnableValueMapping {

		// // 	} else {
		// // 		err := self.doRecurse(ctx, &fd, out, resp, p, typeId)
		// // 		if err != nil {
		// // 			return unwrapError(fmt.Sprintf("converting field %s of MESSAGE %s failed", fd.Name(), fd.Kind()), err)
		// // 		}
		// // 	}

		// // 	if existExceptionField {
		// // 		break
		// // 	}
		// // }
		// // *out = json.EncodeObjectEnd(*out)
		return nil, meta.ErrRead
	default:
		return nil, meta.ErrRead
	}
}
