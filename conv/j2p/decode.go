package j2p

import (
	"encoding/json"
	"fmt"

	"github.com/bytedance/sonic/ast"
	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/dynamicgo/proto/binary"
)

type visitorUserNode interface {
	UserNode()
}

type (
	visitorUserNull    struct{}
	visitorUserBool    struct{ Value bool }
	visitorUserInt64   struct{ Value int64 }
	visitorUserFloat64 struct{ Value float64 }
	visitorUserString  struct{ Value string }
	visitorUserObject  struct{ Value map[string]visitorUserNode }
	visitorUserArray   struct{ Value []visitorUserNode }
)

func (*visitorUserNull) UserNode() {

}

// encode bool for protobuf
func (*visitorUserBool) UserNode() {

}
func (*visitorUserInt64) UserNode() {

}
func (*visitorUserFloat64) UserNode() {

}
func (*visitorUserString) UserNode() {

}
func (*visitorUserObject) UserNode() {

}
func (*visitorUserArray) UserNode() {

}

type visitorUserNodeVisitorDecoder struct {
	stk visitorUserNodeStack
	sp  uint8
}

// keep hierarchy of Array and Object
type visitorUserNodeStack = [256]struct {
	val visitorUserNode
	obj map[string]visitorUserNode
	arr []visitorUserNode
	key string
}

func (self *visitorUserNodeVisitorDecoder) Reset() {
	self.stk = visitorUserNodeStack{}
	self.sp = 0
}

func (self *visitorUserNodeVisitorDecoder) Decode(str string) (visitorUserNode, error) {
	if err := ast.Preorder(str, self, nil); err != nil {
		return nil, err
	}
	return self.result()
}

func (self *visitorUserNodeVisitorDecoder) result() (visitorUserNode, error) {
	if self.sp != 1 {
		return nil, fmt.Errorf("incorrect sp: %d", self.sp)
	}
	return self.stk[0].val, nil
}

func (self *visitorUserNodeVisitorDecoder) incrSP() error {
	self.sp++
	if self.sp == 0 {
		return fmt.Errorf("reached max depth: %d", len(self.stk))
	}
	return nil
}

func (self *visitorUserNodeVisitorDecoder) OnNull() error {
	self.stk[self.sp].val = &visitorUserNull{}
	if err := self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *visitorUserNodeVisitorDecoder) OnBool(v bool) error {
	self.stk[self.sp].val = &visitorUserBool{Value: v}
	if err := self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *visitorUserNodeVisitorDecoder) OnString(v string) error {
	self.stk[self.sp].val = &visitorUserString{Value: v}
	if err := self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *visitorUserNodeVisitorDecoder) OnInt64(v int64, n json.Number) error {
	self.stk[self.sp].val = &visitorUserInt64{Value: v}
	if err := self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *visitorUserNodeVisitorDecoder) OnFloat64(v float64, n json.Number) error {
	self.stk[self.sp].val = &visitorUserFloat64{Value: v}
	if err := self.incrSP(); err != nil {
		return err
	}
	return self.onValueEnd()
}

func (self *visitorUserNodeVisitorDecoder) OnObjectBegin(capacity int) error {
	self.stk[self.sp].obj = make(map[string]visitorUserNode, capacity)
	return self.incrSP()
}

func (self *visitorUserNodeVisitorDecoder) OnObjectKey(key string) error {
	self.stk[self.sp].key = key
	return self.incrSP()
}

func (self *visitorUserNodeVisitorDecoder) OnObjectEnd() error {
	self.stk[self.sp-1].val = &visitorUserObject{Value: self.stk[self.sp-1].obj}
	self.stk[self.sp-1].obj = nil
	return self.onValueEnd()
}

func (self *visitorUserNodeVisitorDecoder) OnArrayBegin(capacity int) error {
	self.stk[self.sp].arr = make([]visitorUserNode, 0, capacity)
	return self.incrSP()
}

func (self *visitorUserNodeVisitorDecoder) OnArrayEnd() error {
	self.stk[self.sp-1].val = &visitorUserArray{Value: self.stk[self.sp-1].arr}
	self.stk[self.sp-1].arr = nil
	return self.onValueEnd()
}

func (self *visitorUserNodeVisitorDecoder) onValueEnd() error {
	if self.sp == 1 {
		return nil
	}
	// [..., Array, Value, sp]
	if self.stk[self.sp-2].arr != nil {
		self.stk[self.sp-2].arr = append(self.stk[self.sp-2].arr, self.stk[self.sp-1].val)
		self.sp--
		return nil
	}
	// [..., Object, ObjectKey, Value, sp]
	self.stk[self.sp-3].obj[self.stk[self.sp-2].key] = self.stk[self.sp-1].val
	self.sp -= 2
	return nil
}

func parseUserNodeRecursive(node visitorUserNode, desc *proto.Descriptor, p *binary.BinaryProtocol) error {
	switch node.(type) {
	case *visitorUserNull:

	case *visitorUserInt64:
		// nhs, ok := node.(*visitorUserInt64)
		// if !ok {
		// 	return meta.NewError(meta.ErrDismatchType, fmt.Sprintf("unsatified visitorUserNodeType : %v"), nil)
		// }
		// fdes, ok := (*desc).(proto.FieldDescriptor)
		// if ok {

		// }
	case *visitorUserFloat64:
		nhs, ok := node.(*visitorUserFloat64)
		if ok {
			fmt.Print(nhs.Value)
		}
	case *visitorUserBool:
		nhs, ok := node.(*visitorUserBool)
		if ok {
			fmt.Print(nhs.Value)
		}
	case *visitorUserString:
		nhs, ok := node.(*visitorUserString)
		if ok {
			fmt.Print(nhs.Value)
		}
	case *visitorUserObject:
		nhs, ok := node.(*visitorUserObject)
		if ok {
			if len(nhs.Value) > 0 {
				fmt.Println("parse Object start")
				for k, v := range nhs.Value {
					fmt.Print(k)
					if v != nil {
						parseUserNodeRecursive(v, desc, p)
					}
				}
			}
		}
	case *visitorUserArray:
		nhs, ok := node.(*visitorUserArray)
		if ok {
			if len(nhs.Value) > 0 {
				for _, v := range nhs.Value {
					fmt.Println("parse Array start")
					if v != nil {
						parseUserNodeRecursive(v, desc, p)
					}
				}
			}
		}
	default:
		fmt.Println("unknown field")
	}
	return nil
}
