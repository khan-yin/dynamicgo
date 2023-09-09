package binary

import (
	"context"
	"fmt"
	"testing"

	"github.com/cloudwego/dynamicgo/proto"
	"github.com/cloudwego/dynamicgo/testdata/pb/testprotos"
	goproto "google.golang.org/protobuf/proto"
)

type TestMode int8

const (
	PartialTest TestMode = iota
	NestedTest
)

func BinaryDataBuild(mode TestMode) []byte {
	switch mode {
	case PartialTest:
		req := &testprotos.ExampleParitalReq{}
		req.Msg = "firstTest"
		cookie := 4.24
		req.Cookie = &cookie
		req.Status = 2
		req.Header = true
		req.Code = 432
		data, err := goproto.Marshal(req)
		if err != nil {
			panic("proto Marshal failed: PartialTest")
		}
		return data
	}
	return nil
}

func TestBinaryProtocol_ReadAnyWithDesc(t *testing.T) {
	p1, err := proto.NewDescritorFromPath(context.Background(), "../../testdata/idl/example.proto")
	if err != nil {
		panic(err)
	}
	// test Partial
	partialFieldDesc := (*p1).Methods().ByName("PartialMethod").Output().Fields().ByNumber(1)

	data := BinaryDataBuild(PartialTest)
	p := NewBinaryProtol(data)
	v, err := p.ReadAnyWithDesc(&partialFieldDesc, false, false, true)
	fmt.Printf("%#v", v)
	//p = NewBinaryProtocolBuffer()

}
