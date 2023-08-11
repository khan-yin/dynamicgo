package base

import (
	"encoding/binary"
	"math"
	"reflect"
	"sync"
	"unicode/utf8"
	"unsafe"

	"github.com/cloudwego/dynamicgo/internal/primitive"
	"github.com/cloudwego/dynamicgo/internal/rt"
	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/dynamicgo/proto/errors"
	"github.com/cloudwego/dynamicgo/proto/protowire"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const (
	defaultBufferSize = 4096
	growBufferFactor  = 1
)

var byteType = rt.UnpackType(reflect.TypeOf(byte(0)))

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

// WriteBool
func (p *BinaryProtocol) WriteBool(value bool) error {
	if value {
		return p.WriteUI64(uint64(1))
	} else {
		return p.WriteUI64(uint64(0))
	}
}

// WriteInt32
func (p *BinaryProtocol) WriteI32(value int32) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeInt32(p.Buf, value)
	return nil
}

// WriteSint32
func (p *BinaryProtocol) WriteSI32(value int32) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeSint32(p.Buf, value)
	return nil
}

// WriteUint32
func (p *BinaryProtocol) WriteUI32(value uint32) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeUint32(p.Buf, value)
	return nil
}

// Writefixed32
func (p *BinaryProtocol) Writefixed32(value int32) error {
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
func (p *BinaryProtocol) WriteSI64(value int64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeSint64(p.Buf, value)
	return nil
}

// WriteUint64
func (p *BinaryProtocol) WriteUI64(value uint64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeUint64(p.Buf, value)
	return nil
}

// Writefixed64
func (p *BinaryProtocol) Writefixed64(value uint64) error {
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
		return errors.InvalidUTF8(value)
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
func (p *BinaryProtocol) WriteList() error {
	return nil
}

/**
 * WriteMap
 */
func (p *BinaryProtocol) WriteMap() error {
	return nil
}

/**
 * Write Message
 */
func (p *BinaryProtocol) WriteMessageSlow() error {
	return nil
}

// When encoding length-prefixed fields, we speculatively set aside some number of bytes
// for the length, encode the data, and then encode the length (shifting the data if necessary
// to make room).
const speculativeLength = 1

func appendSpeculativeLength(b []byte) int {
	pos := len(b)
	b = append(b, "\x00\x00\x00\x00"[:speculativeLength]...)
	return pos
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
	// protowire.AppendVarint(b[:pos], uint64(mlen))
	return b
}

// WriteAnyWithDesc
func (p *BinaryProtocol) WriteAnyWithDesc(desc proto.FieldDescriptor, val interface{}, cast bool, disallowUnknown bool, useFieldName bool) error {
	switch {
	case desc.IsList():
		return p.WriteList()
	case desc.IsMap():
		return p.WriteMap()
	default:
		if e := p.AppendTag(proto.Number(desc.Number()), wireTypes[desc.Kind()]); e != nil {
			return errors.New("AppendTag failed : %v", e.Error())
		}
		return p.WriteBaseTypeWithDesc(desc, val)
	}
}

// WriteBaseType with desc, not thread safe
func (p *BinaryProtocol) WriteBaseTypeWithDesc(fd proto.FieldDescriptor, val interface{}) error {
	switch fd.Kind() {
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
			return errors.New("WriteEnum error")
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
		p.WriteSI32(v)
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
		p.WriteUI32(v)
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
		p.WriteSI64(v)
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
		p.WriteUI64(v)
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
		p.Writefixed32(v)
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
			return errors.InvalidUTF8(string(fd.FullName()))
		}
		p.WriteString(v)
	case protoreflect.Kind(proto.BytesKind):
		v, ok := val.([]byte)
		if !ok {
			return errors.New("WriteBytesType error")
		}
		p.WriteBytes(v)
	// case protoreflect.Kind(proto.MessageKind):
	// 	var pos int
	// 	var err error
	// 	pos = appendSpeculativeLength(p.Buf)
	// 	b, err = o.marshalMessage()
	// 	if err != nil {
	// 		return b, err
	// 	}
	// 	b = finishSpeculativeLength(b, pos)
	// case protoreflect.Kind(proto.GroupKind):
	// 	var err error
	// 	b, err = o.marshalMessage(b, v.Message())
	// 	if err != nil {
	// 		return b, err
	// 	}
	// 	b = protowire.AppendVarint(b, protowire.EncodeTag(fd.Number(), protowire.EndGroupType))
	default:
		return errors.New("invalid kind %v", fd.Kind())
	}
	return nil
}
