package j2p

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"unsafe"

	"github.com/bytedance/sonic/ast"
	"github.com/chenzhuoyu/base64x"
	"github.com/cloudwego/dynamicgo/internal/rt"
	"github.com/cloudwego/dynamicgo/meta"
	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/dynamicgo/proto/binary"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// memory resize factor
const (
	// new = old + old >> growSliceFactor
	growStkFactory = 1

	defaultStkDepth = 128
)

var (
	vuPool = sync.Pool{
		New: func() interface{} {
			return &VisitorUserNode{
				sp:  0,
				p:   binary.NewBinaryProtocolBuffer(),
				stk: make([]VisitorUserNodeStack, defaultStkDepth),
			}
		},
	}
)

// NewVisitorUserNode get a new VisitorUserNode from sync.Pool
func NewVisitorUserNode(buf []byte) *VisitorUserNode {
	vu := vuPool.Get().(*VisitorUserNode)
	vu.p.Buf = buf
	return vu
}

// NewVisitorUserNode gets a new VisitorUserNode from sync.Pool
// and reuse the buffer in pool
func NewVisitorUserNodeBuffer() *VisitorUserNode {
	vu := vuPool.Get().(*VisitorUserNode)
	return vu
}

// FreeVisitorUserNode resets the buffer and puts the VisitorUserNode back to sync.Pool
func FreeVisitorUserNodePool(vu *VisitorUserNode) {
	vu.Reset()
	vuPool.Put(vu)
}

// Recycle put the VisitorUserNode back to sync.Pool
func (self *VisitorUserNode) Recycle() {
	self.Reset()
	vuPool.Put(self)
}

// Reset resets the buffer and read position
func (self *VisitorUserNode) Reset() {
	self.sp = 0
	self.p.Reset()
	for i := 0; i < len(self.stk); i++ {
		self.stk[i].Reset()
	}
}

/** p use to encode pbEncode
 *  desc represent fieldDescriptor
 *  pos is used when encode message\mapValue\unpackedList
 */
type VisitorUserNode struct {
	stk []VisitorUserNodeStack
	sp  uint8
	p   *binary.BinaryProtocol
}

// keep hierarchy of Array and Object, arr represent current is List, obj represent current is Map/Object
type VisitorUserNodeStack struct {
	obj   bool
	arr   bool
	key   string
	state visitorUserNodeState
}

func (stk VisitorUserNodeStack) Reset() {
	stk.obj = false
	stk.arr = false
	stk.key = ""
	stk.state.lenPos = -1
	stk.state.desc = nil
}

// record descriptor、preWrite lenPos
type visitorUserNodeState struct {
	desc   *proto.Descriptor
	lenPos int
}

func (self *VisitorUserNode) Decode(bytes []byte, desc *proto.Descriptor) ([]byte, error) {
	// init initial visitorUserNodeState
	self.stk[self.sp].state = visitorUserNodeState{desc: desc, lenPos: -1}
	str := rt.Mem2Str(bytes)
	if err := ast.Preorder(str, self, nil); err != nil {
		return nil, err
	}
	return self.result()
}

func (self *VisitorUserNode) result() ([]byte, error) {
	if self.sp != 1 {
		return nil, fmt.Errorf("incorrect sp: %d", self.sp)
	}
	return self.p.RawBuf(), nil
}

func (self *VisitorUserNode) incrSP() error {
	self.sp++
	if self.sp == 0 {
		return fmt.Errorf("reached max depth: %d", len(self.stk))
	}
	return nil
}

func (self *VisitorUserNode) OnNull() error {
	// self.stk[self.sp].val = &visitorUserNull{}
	if err := self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *VisitorUserNode) OnBool(v bool) error {
	var err error
	curDesc := self.stk[self.SearchPrevStateNodeIndex()].state.desc
	convertDesc := (*curDesc).(proto.FieldDescriptor)
	if err = self.p.AppendTagByDesc(convertDesc); err != nil {
		return err
	}
	if err = self.p.WriteBool(v); err != nil {
		return err
	}
	if err = self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *VisitorUserNode) OnString(v string) error {
	var err error
	curDesc := self.stk[self.SearchPrevStateNodeIndex()].state.desc
	convertDesc := (*curDesc).(proto.FieldDescriptor)
	// convert string、bytesType
	switch convertDesc.Kind() {
	case proto.BytesKind:
		// bytesData, err := base64.StdEncoding.DecodeString(v)
		bytesData, err := base64x.StdEncoding.DecodeString(v)
		if err = self.p.AppendTagByDesc(convertDesc); err != nil {
			return err
		}
		if err = self.p.WriteBytes(bytesData); err != nil {
			return err
		}
	case proto.StringKind:
		if err = self.p.AppendTagByDesc(convertDesc); err != nil {
			return err
		}
		if err = self.p.WriteString(v); err != nil {
			return err
		}
	default:
		return newError(meta.ErrDismatchType, "param isn't stringType", nil)
	}
	if err = self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *VisitorUserNode) OnInt64(v int64, n json.Number) error {
	var err error
	curDesc := self.stk[self.SearchPrevStateNodeIndex()].state.desc
	convertDesc := (*curDesc).(proto.FieldDescriptor)
	switch convertDesc.Kind() {
	case proto.Int32Kind, proto.Sint32Kind, proto.Sfixed32Kind, proto.Fixed32Kind:
		convertData := *(*int32)(unsafe.Pointer(&v))
		if !convertDesc.IsList() {
			if err = self.p.AppendTagByDesc(convertDesc); err != nil {
				return err
			}
		}
		if err = self.p.WriteI32(convertData); err != nil {
			return err
		}
	case proto.Uint32Kind:
		convertData := *(*uint32)(unsafe.Pointer(&v))
		if !convertDesc.IsList() {
			if err = self.p.AppendTagByDesc(convertDesc); err != nil {
				return err
			}
		}
		if err = self.p.WriteUint32(convertData); err != nil {
			return err
		}
	case proto.Uint64Kind:
		convertData := *(*uint64)(unsafe.Pointer(&v))
		if !convertDesc.IsList() {
			if err = self.p.AppendTagByDesc(convertDesc); err != nil {
				return err
			}
		}
		if err = self.p.WriteUint64(convertData); err != nil {
			return err
		}
	case proto.Int64Kind:
		if !convertDesc.IsList() {
			if err = self.p.AppendTagByDesc(convertDesc); err != nil {
				return err
			}
		}
		if err = self.p.WriteI64(v); err != nil {
			return err
		}
	default:
		return newError(meta.ErrDismatchType, "param isn't intType", nil)
	}
	if err = self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *VisitorUserNode) OnFloat64(v float64, n json.Number) error {
	var err error
	curDesc := self.stk[self.SearchPrevStateNodeIndex()].state.desc
	convertDesc := (*curDesc).(proto.FieldDescriptor)
	switch convertDesc.Kind() {
	case proto.FloatKind:
		convertData := *(*float32)(unsafe.Pointer(&v))
		if err = self.p.AppendTagByDesc(convertDesc); err != nil {
			return err
		}
		if err = self.p.WriteFloat(convertData); err != nil {
			return err
		}
	case proto.DoubleKind:
		convertData := *(*float64)(unsafe.Pointer(&v))
		if err = self.p.AppendTagByDesc(convertDesc); err != nil {
			return err
		}
		if err = self.p.WriteDouble(convertData); err != nil {
			return err
		}
	default:
		return newError(meta.ErrDismatchType, "param isn't floatType", nil)
	}
	// self.stk[self.sp].val = &visitorUserFloat64{Value: v}
	if err = self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *VisitorUserNode) OnObjectBegin(capacity int) error {
	self.stk[self.sp].obj = true
	lastDescIdx := self.SearchPrevStateNodeIndex()
	if lastDescIdx > 0 {
		var elemLenPos int
		objDesc := self.stk[lastDescIdx].state.desc
		convertDesc := (*objDesc).(proto.FieldDescriptor)
		// case List<Message>/Map<xxx,Message>, insert MessageTag、MessageLen, store innerMessageDesc into stack
		if convertDesc != nil && convertDesc.Message() != nil {
			parentMsg := convertDesc.ContainingMessage()
			if convertDesc.IsList() || parentMsg.IsMapEntry() {
				innerElemDesc := (*objDesc).(proto.FieldDescriptor)
				if err := self.p.AppendTag(proto.Number(innerElemDesc.Number()), proto.BytesType); err != nil {
					return meta.NewError(meta.ErrWrite, "append prefix tag failed", nil)
				}
				self.p.Buf, elemLenPos = binary.AppendSpeculativeLength(self.p.Buf)
				self.stk[self.sp].state = visitorUserNodeState{
					desc:   objDesc,
					lenPos: elemLenPos,
				}
			}
		}
	}
	return self.incrSP()
}

func (self *VisitorUserNode) OnObjectKey(key string) error {
	self.stk[self.sp].key = key
	var rootDesc proto.MessageDescriptor
	var curDesc proto.FieldDescriptor
	var preNodeState visitorUserNodeState
	// search preNodeState
	i := self.SearchPrevStateNodeIndex()
	preNodeState = self.stk[i].state

	// recognize descriptor type
	if i == 0 {
		rootDesc = ((*preNodeState.desc).(proto.MessageDescriptor))
	} else {
		curDesc = ((*preNodeState.desc).(proto.FieldDescriptor))
	}
	// initial hierarchy
	if rootDesc != nil {
		curDesc = rootDesc.Fields().ByJSONName(key)
		// complex structure, get inner fieldDesc by jsonname
	} else if curDesc != nil && curDesc.Message() != nil {
		if curDesc.IsList() && !curDesc.IsPacked() {
			// case Unpacked List
			if fieldDesc := curDesc.Message().Fields().ByJSONName(key); fieldDesc != nil {
				curDesc = fieldDesc
			}
		} else if curDesc.IsMap() {
			// case Map
		} else if !curDesc.IsPacked() {
			// case Message
			curDesc = curDesc.Message().Fields().ByJSONName(key)
		}
	}

	if curDesc != nil {
		curNodeLenPos := -1
		convertDescriptor := curDesc.(proto.Descriptor)
		// case PackedList/Message, need to write prefix Tag、Len, and store to stack
		if (curDesc.IsList() && curDesc.IsPacked()) || (curDesc.Kind() == proto.MessageKind && !curDesc.IsMap() && !curDesc.IsList()) {
			if err := self.p.AppendTag(proto.Number(curDesc.Number()), proto.BytesType); err != nil {
				return meta.NewError(meta.ErrWrite, "append prefix tag failed", nil)
			}
			self.p.Buf, curNodeLenPos = binary.AppendSpeculativeLength(self.p.Buf)
		} else {
			// mapkey but not mapName, write pairTag、pairLen、mapKeyTag、mapKeyLen、mapData, may have bug here
			if curDesc.IsMap() && curDesc.JSONName() != key {
				// pay attention to Key type conversion
				var keyData interface{}
				mapKeyDesc := curDesc.MapKey()
				switch mapKeyDesc.Kind() {
				case protoreflect.Kind(proto.Int32Kind), protoreflect.Kind(proto.Sint32Kind), protoreflect.Kind(proto.Sfixed32Kind), protoreflect.Kind(proto.Fixed32Kind):
					t, _ := strconv.ParseInt(key, 10, 32)
					keyData = int32(t)
				case protoreflect.Kind(proto.Uint32Kind):
					t, _ := strconv.ParseInt(key, 10, 32)
					keyData = uint32(t)
				case protoreflect.Kind(proto.Uint64Kind):
					t, _ := strconv.ParseInt(key, 10, 64)
					keyData = uint64(t)
				case protoreflect.Kind(proto.Int64Kind):
					t, _ := strconv.ParseInt(key, 10, 64)
					keyData = t
				case protoreflect.Kind(proto.BoolKind):
					t, _ := strconv.ParseBool(key)
					keyData = t
				default:
					keyData = key
				}
				self.p.AppendTag(curDesc.Number(), proto.BytesType)
				self.p.Buf, curNodeLenPos = binary.AppendSpeculativeLength(self.p.Buf)
				if err := self.p.WriteAnyWithDesc(&mapKeyDesc, keyData, false, false, true); err != nil {
					return err
				}
				// store MapValueDesc into stack
				convertDescriptor = curDesc.MapValue().(proto.Descriptor)
			}
		}
		// store fieldDesc into stack
		self.stk[self.sp].state = visitorUserNodeState{
			desc:   &convertDescriptor,
			lenPos: curNodeLenPos,
		}
	}
	return self.incrSP()
}

// search the last StateNode which desc isn't empty
func (self *VisitorUserNode) SearchPrevStateNodeIndex() int {
	var i int
	for i = int(self.sp); i >= 0; i-- {
		if i == 0 || self.stk[i].state.desc != nil {
			break
		}
	}
	return i
}

func (self *VisitorUserNode) OnObjectEnd() error {
	self.stk[self.sp-1].obj = false

	// fill prefix_length when MessageEnd, may have problem with rootDesc
	parentDescIdx := self.SearchPrevStateNodeIndex()
	if parentDescIdx > 0 {
		parentNodeState := self.stk[parentDescIdx].state
		if parentNodeState.desc != nil && parentNodeState.lenPos != -1 {
			self.p.Buf = binary.FinishSpeculativeLength(self.p.Buf, parentNodeState.lenPos)
			self.stk[parentDescIdx].state.desc = nil
			self.stk[parentDescIdx].state.lenPos = -1
		}
	}
	return self.onValueEnd()
}

func (self *VisitorUserNode) OnArrayBegin(capacity int) error {
	self.stk[self.sp].arr = true
	return self.incrSP()
}

func (self *VisitorUserNode) OnArrayEnd() error {
	self.stk[self.sp-1].arr = false

	// case arrayEnd, fill arrayPrefixLen
	var parentNodeState visitorUserNodeState
	if self.sp >= 2 {
		parentNodeState = self.stk[self.sp-2].state
		if parentNodeState.lenPos != -1 {
			// case Packedlist, fill prefixLength
			self.p.Buf = binary.FinishSpeculativeLength(self.p.Buf, parentNodeState.lenPos)
			self.stk[self.sp-2].state.desc = nil
			self.stk[self.sp-2].state.lenPos = -1
		}
	}
	return self.onValueEnd()
}

func (self *VisitorUserNode) onValueEnd() error {
	if self.sp == 1 {
		return nil
	}
	// [..., Array, Value, sp]
	if self.stk[self.sp-2].arr != false {
		// self.stk[self.sp-2].arr = append(self.stk[self.sp-2].arr, self.stk[self.sp-1].val)
		self.sp--
		return nil
	}
	// [..., Object, ObjectKey, Value, sp]
	if self.stk[self.sp-3].obj != false {
		// self.stk[self.sp-3].obj[self.stk[self.sp-2].key] = self.stk[self.sp-1].val
		self.sp -= 2

		// case MapValueEnd, fill pairLength, need to exclude Message[Message]
		if self.sp >= 2 {
			parentNodeState := self.stk[self.sp-2].state
			if parentNodeState.desc != nil {
				if (*parentNodeState.desc).(proto.FieldDescriptor).IsMap() {
					// case mapValueEnd, fill pairLength
					self.p.Buf = binary.FinishSpeculativeLength(self.p.Buf, self.stk[self.sp].state.lenPos)
				}
			}
		}

		self.stk[self.sp].state.desc = nil
		self.stk[self.sp].state.lenPos = -1
	}
	return nil
}
