package p2j

import (
	"context"

	"github.com/cloudwego/dynamicgo/conv"
	"github.com/cloudwego/dynamicgo/http"
	"github.com/cloudwego/dynamicgo/meta"
	"github.com/cloudwego/dynamicgo/proto"
)

type BinaryConv struct {
	opts conv.Options
}

// NewBinaryConv returns a new BinaryConv
func NewBinaryConv(opts conv.Options) BinaryConv {
	return BinaryConv{opts: opts}
}

// SetOptions sets options
func (self *BinaryConv) SetOptions(opts conv.Options) {
	self.opts = opts
}

// Do converts protobuf binary (pbytes) to json bytes (jbytes)
// desc is the protobuf type descriptor of the protobuf binary, usually it is a response Message type
func (self *BinaryConv) Do(ctx context.Context, desc *proto.TypeDescriptor, pbytes []byte) (json []byte, err error) {
	buf := conv.NewBytes()

	// resp alloc but not use
	var resp http.ResponseSetter
	if self.opts.EnableHttpMapping {
		respi := ctx.Value(conv.CtxKeyHTTPResponse)
		if respi != nil {
			respi, ok := respi.(http.ResponseSetter)
			if !ok {
				return nil, wrapError(meta.ErrInvalidParam, "invalid http.Response", nil)
			}
			resp = respi
		} else {
			return nil, wrapError(meta.ErrInvalidParam, "no http response in context", nil)
		}
	}

	err = self.do(ctx, pbytes, desc, buf, resp)
	if err == nil && len(*buf) > 0 {
		json = make([]byte, len(*buf))
		copy(json, *buf)
	}

	conv.FreeBytes(buf)
	return
}

// DoInto behaves like Do, but it writes the result to buffer directly instead of returning a new buffer
func (self *BinaryConv) DoInto(ctx context.Context, desc *proto.TypeDescriptor, pbytes []byte, buf *[]byte) (err error) {
	// resp alloc but not use
	var resp http.ResponseSetter
	if self.opts.EnableHttpMapping {
		respi := ctx.Value(conv.CtxKeyHTTPResponse)
		if respi != nil {
			respi, ok := respi.(http.ResponseSetter)
			if !ok {
				return wrapError(meta.ErrInvalidParam, "invalid http.Response", nil)
			}
			resp = respi
		} else {
			return wrapError(meta.ErrInvalidParam, "no http response in context", nil)
		}
	}

	return self.do(ctx, pbytes, desc, buf, resp)
}

func TestNestedField(t *testing.T) {
	protoContent := `
	syntax = "proto3";
	message Item {
		repeated string ids = 1;
	}

	message ExampleReq {
		repeated Item items = 1;
	}

	message ExampleResp {
		repeated Item items = 1;
	}

	service Service {
		rpc Example(ExampleReq) returns (ExampleResp);
	}
	`
	dataContent := `{"items":[{"ids":["123"]},{"ids":["456"]}]}`
	// itemtag L [ idstag1 L V idstag1 L V ] itemtag L [ idstag1 L V ]

	req := &examplepb.NestedExampleReq{
		Items: []*examplepb.Item{
			{Ids: []string{"123"}},
			{Ids: []string{"456"}},
		},
	}

	t.Run("nested string", func(t *testing.T) {
		ctx := context.Background()
		serviceDesc, err := proto.NewDescritorFromContent(ctx, "example.proto", protoContent, map[string]string{})
		if err != nil {
			log.Fatal(err)
		}
		reqDesc := serviceDesc.LookupMethodByName("Example").Input()
		conv1 := j2p.NewBinaryConv(conv.Options{})
		origindata, err := goproto.Marshal(req)
		if err != nil {
			t.Fatal("marshal error:", err)
		}
		reqProto, err := conv1.Do(ctx, reqDesc, []byte(dataContent))
		fmt.Println(len(origindata), len(reqProto))
		if err != nil {
			t.Fatal("json marshal to protobuf error")
		}
		conv2 := NewBinaryConv(conv.Options{})
		reqJson, err := conv2.Do(ctx, reqDesc, reqProto)
		if err != nil {
			t.Fatal("protobuf marshal to json error")
		}

		require.Equal(t, dataContent, string(reqJson))

	})
}
