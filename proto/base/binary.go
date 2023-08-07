package base

import (
	"encoding/binary"
	"errors"
	"math"
	"reflect"
	"sync"
	"unicode/utf8"
	"unsafe"

	"github.com/cloudwego/dynamicgo/internal/rt"
	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/dynamicgo/proto/protowire"
)

const (
	defaultBufferSize = 4096
	growBufferFactor  = 1
)

var byteType = rt.UnpackType(reflect.TypeOf(byte(0)))

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
func (p BinaryProtocol) RawBuf() []byte {
	return p.Buf
}

// Left returns the left bytes to read
func (p BinaryProtocol) Left() int {
	return len(p.Buf) - p.Read
}

// WriteBool
func (p BinaryProtocol) WriteBool(value bool) error {
	if value {
		return p.WriteUI64(uint64(1))
	} else {
		return p.WriteUI64(uint64(0))
	}
}

// WriteInt32
func (p BinaryProtocol) WriteI32(value int64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeInt32(p.Buf, int32(value))
	return nil
}

// WriteSint32
func (p BinaryProtocol) WriteSI32(value int64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeSint32(p.Buf, int32(value))
	return nil
}

// WriteUint32
func (p BinaryProtocol) WriteUI32(value uint64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeUint32(p.Buf, uint32(value))
	return nil
}

// Writefixed32
func (p BinaryProtocol) Writefixed32(value uint64) error {
	v, err := p.malloc(4)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(v, uint32(value))
	return err
}

// WriteSfixed32
func (p BinaryProtocol) WriteSfixed32(value int64) error {
	v, err := p.malloc(4)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(v, uint32(value))
	return err
}

// WriteInt64
func (p BinaryProtocol) WriteI64(value int64) error {
	protowire.BinaryEncoder{}.EncodeInt64(p.Buf, value)
	return nil
}

// WriteSint64
func (p BinaryProtocol) WriteSI64(value int64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeSint64(p.Buf, value)
	return nil
}

// WriteUint64
func (p BinaryProtocol) WriteUI64(value uint64) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeUint64(p.Buf, value)
	return nil
}

// Writefixed64
func (p BinaryProtocol) Writefixed64(value uint64) error {
	v, err := p.malloc(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(v, value)
	return err
}

// WriteSfixed64
func (p BinaryProtocol) WriteSfixed64(value int64) error {
	v, err := p.malloc(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(v, uint64(value))
	return err
}

// WriteFloat
func (p BinaryProtocol) WriteFloat(value float64) error {
	v, err := p.malloc(4)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(v, math.Float32bits(float32(value)))
	return err
}

// WriteDouble
func (p BinaryProtocol) WriteDouble(value float64) error {
	v, err := p.malloc(8)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint64(v, math.Float64bits(value))
	return err
}

// WriteString
func (p BinaryProtocol) WriteString(value string) error {
	if !utf8.ValidString(value) {
		return proto.InvalidUTF8(value)
	}
	p.Buf = protowire.BinaryEncoder{}.EncodeString(p.Buf, value)
	return nil
}

// WriteBytes
func (p BinaryProtocol) WriteBytes(value []byte) error {
	p.Buf = protowire.BinaryEncoder{}.EncodeBytes(p.Buf, value)
	return nil
}
