package base

import (
	"io"

	"github.com/cloudwego/dynamicgo/proto"
)

var typeSize = [256]int{
	proto.NIL:	-1, // nil
	proto.DOUBLE:	-1,
	proto.FLOAT:  -1,
	proto.INT64:  -1,
	proto.UINT64: -1,
	proto.INT32:  -1,
	proto.FIX64:  -1,
	proto.FIX32:  -1,
	proto.BOOL:   -1,
	proto.STRING: -1,
	proto.GROUP:  -1, // deprecated
	proto.MESSAGE: -1,
	proto.BYTE:   -1,
	proto.UINT32: -1,
	proto.ENUM:   -1,
	proto.SFIX32: -1,
	proto.SFIX64: -1,
	proto.SINT32: -1,
	proto.SINT64: -1,
	proto.LIST:   -1,
	proto.MAP:	-1,
}

const MaxSkipDepth = 1023

func (p *BinaryProtocol) Skip(fieldType proto.Type, useNative bool) (err error) {
	return p.SkipGo(fieldType, MaxSkipDepth)
}

// TypeSize returns the size of the given type.
// -1 means variable size (LIST, SET, MAP, STRING)
// 0 means unknown type
func TypeSize(t proto.Type) int {
	return typeSize[t]
}

// SkipGo skips over the value for the given type using Go implementation.
func (p *BinaryProtocol) SkipGo(fieldType proto.Type, maxDepth int) (err error) {
	if maxDepth <= 0 {
		return errExceedDepthLimit
	}
	switch fieldType {
	case proto.DOUBLE:
		_, err = p.ReadDouble()
		return
	case proto.FLOAT:
		_, err = p.ReadFloat()
		return
	case proto.INT64:
		_, err = p.ReadI64()
		return
	case proto.INT32:
		_, err = p.ReadI32()
		return
	case proto.FIX64:
		_, err = p.ReadFixed64()
		return
	case proto.FIX32:
		_, err = p.ReadFixed32()
		return
	case proto.BOOL:
		_, err = p.ReadBool()
		return
	case proto.STRING:
		_, err = p.ReadString(false)
		return
	case proto.BYTE:
		_, err = p.ReadBytes()
		return
	case proto.UINT32:
		_, err = p.ReadUint32()
		return
	case proto.ENUM:
		_, err = p.ReadI32()
		return
	case proto.UINT64:
		_, err = p.ReadUint64()
		return
	case proto.SFIX32:
		_, err = p.ReadSfixed32()
		return
	case proto.SFIX64:
		_, err = p.ReadSfixed64()
		return
	case proto.SINT32:
		_, err = p.ReadSint32()
		return
	case proto.SINT64:
		_, err = p.ReadSint64()
		return
	case proto.LIST:
		
	case proto.MAP:

	case proto.MESSAGE:
		// if _, err = p.ReadStructBegin(); err != nil {
		// 	return err
		// }
		for {
			_, typeId, _, _ := p.ReadFieldBegin()
			if typeId == STOP {
				break
			}
			//fastpath
			if n := typeSize[typeId]; n > 0 {
				p.Read += n
				if p.Read > len(p.Buf) {
					return io.EOF
				}
				continue
			}
			err := p.SkipGo(typeId, maxDepth-1)
			if err != nil {
				return err
			}
			p.ReadFieldEnd()
		}
		return p.ReadStructEnd()
	case MAP:
		keyType, valueType, size, err := p.ReadMapBegin()
		if err != nil {
			return err
		}
		//fastpath
		if k, v := typeSize[keyType], typeSize[valueType]; k > 0 && v > 0 {
			p.Read += (k + v) * size
			if p.Read > len(p.Buf) {
				return io.EOF
			}
		} else {
			if size > len(p.Buf)-p.Read {
				return errInvalidDataSize
			}
			for i := 0; i < size; i++ {
				err := p.SkipGo(keyType, maxDepth-1)
				if err != nil {
					return err
				}
				err = p.SkipGo(valueType, maxDepth-1)
				if err != nil {
					return err
				}
			}
		}
		return p.ReadMapEnd()
	case SET, LIST:
		elemType, size, err := p.ReadListBegin()
		if err != nil {
			return err
		}
		//fastpath
		if v := typeSize[elemType]; v > 0 {
			p.Read += v * size
			if p.Read > len(p.Buf) {
				return io.EOF
			}
		} else {
			if size > len(p.Buf)-p.Read {
				return errInvalidDataSize
			}
			for i := 0; i < size; i++ {
				err := p.SkipGo(elemType, maxDepth-1)
				if err != nil {
					return err
				}
			}
		}
		return p.ReadListEnd()
	default:
		return
	}
}