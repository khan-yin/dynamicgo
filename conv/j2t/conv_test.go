/**
 * Copyright 2023 CloudWeGo Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package j2t

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	stdh "net/http"
	"net/url"
	"os"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic/ast"
	sjson "github.com/bytedance/sonic/ast"
	"github.com/cloudwego/dynamicgo/conv"
	"github.com/cloudwego/dynamicgo/http"
	"github.com/cloudwego/dynamicgo/meta"
	ebase "github.com/cloudwego/dynamicgo/testdata/kitex_gen/base"
	"github.com/cloudwego/dynamicgo/testdata/kitex_gen/example3"
	"github.com/cloudwego/dynamicgo/testdata/kitex_gen/null"
	"github.com/cloudwego/dynamicgo/testdata/sample"
	"github.com/cloudwego/dynamicgo/thrift"
	"github.com/cloudwego/dynamicgo/thrift/annotation"
	"github.com/cloudwego/dynamicgo/thrift/base"
	"github.com/stretchr/testify/require"
)

var (
	debugAsyncGC = os.Getenv("SONIC_NO_ASYNC_GC") == ""
)

func TestMain(m *testing.M) {
	go func() {
		if !debugAsyncGC {
			return
		}
		println("Begin GC looping...")
		for {
			runtime.GC()
			debug.FreeOSMemory()
		}
	}()
	time.Sleep(time.Millisecond)
	annotation.InitAGWAnnos()
	m.Run()
}

const (
	exampleIDLPath = "../../testdata/idl/example3.thrift"
	nullIDLPath    = "../../testdata/idl/null.thrift"
	exampleJSON    = "../../testdata/data/example3req.json"
	nullJSON       = "../../testdata/data/null_pass.json"
	nullerrJSON    = "../../testdata/data/null_err.json"
)

func TestCases(t *testing.T) {
	var tests = []struct {
		name     string
		idl      string
		includes map[string]string
		js       string
		opt      conv.Options
		want     interface{}
		err      error
	}{
		{
			name: "int2double_vm",
			idl: `struct Req {
	1: optional double body (api.js_conv=""), 
}

service SVR {
	void Method(1: Req req)
}	
`,
			includes: nil,
			js:       `{"body":"-1"}`,
			opt:      conv.Options{},
			want: map[string]interface{}{
				"body": float64(-1),
			},
		},
		{
			name: "int2double",
			idl: `struct Req {
	1: optional double body, 
}

service SVR {
	void Method(1: Req req)
}	
`,
			includes: nil,
			js:       `{"body":-2}`,
			opt:      conv.Options{EnableValueMapping: true},
			want: map[string]interface{}{
				"body": float64(-2),
			},
		},
	}
	for _, c := range tests {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			svc, err := thrift.NewDefaultOptions().NewDescritorFromContent(ctx, "a.thrift", c.idl, c.includes, false)
			if err != nil {
				t.Fatal(err)
			}
			desc := svc.Functions()["Method"].Request().Struct().FieldById(1).Type()
			cv := NewBinaryConv(conv.Options{
				EnableValueMapping: true,
			})
			out, err := cv.Do(ctx, desc, []byte(c.js))
			if err != nil {
				if c.err == nil || c.err.Error() == err.Error() {
					t.Fatal(err)
				}
			}
			v, err := thrift.NewBinaryProtocol(out).ReadAnyWithDesc(desc, false, false, false, true)
			if err != nil {
				if c.err == nil || c.err.Error() == err.Error() {
					t.Fatal(err)
				}
			}
			require.Equal(t, c.want, v)
		})
	}
}

func TestConvJSON2Thrift(t *testing.T) {
	desc := getExampleDesc()
	data := getExampleData()
	cv := NewBinaryConv(conv.Options{})
	ctx := context.Background()
	out, err := cv.Do(ctx, desc, data)
	require.Nil(t, err)
	exp := example3.NewExampleReq()
	err = json.Unmarshal(data, exp)
	require.Nil(t, err)
	act := example3.NewExampleReq()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	require.Equal(t, exp, act)
}

func TestConvHTTP2Thrift(t *testing.T) {
	desc := getExampleDesc()
	data := getExampleData()
	exp := example3.NewExampleReq()
	err := json.Unmarshal(data, exp)
	require.Nil(t, err)
	req := getExampleReq(exp, true, data)
	cv := NewBinaryConv(conv.Options{
		EnableHttpMapping: true,
	})
	ctx := context.Background()
	ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
	out, err := cv.Do(ctx, desc, data)
	require.NoError(t, err)

	act := example3.NewExampleReq()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	require.Equal(t, exp, act)
}

func getExampleDesc() *thrift.TypeDescriptor {
	opts := thrift.Options{}
	svc, err := opts.NewDescritorFromPath(context.Background(), exampleIDLPath)
	if err != nil {
		panic(err)
	}
	return svc.Functions()["ExampleMethod"].Request().Struct().FieldById(1).Type()
}

func getErrorExampleDesc() *thrift.TypeDescriptor {
	opts := thrift.Options{}
	svc, err := opts.NewDescritorFromPath(context.Background(), exampleIDLPath)
	if err != nil {
		panic(err)
	}
	return svc.Functions()["ErrorMethod"].Request().Struct().FieldById(1).Type()
}

func getExampleInt2FloatDesc() *thrift.TypeDescriptor {
	opts := thrift.Options{}
	svc, err := opts.NewDescritorFromPath(context.Background(), exampleIDLPath)
	if err != nil {
		panic(err)
	}
	return svc.Functions()["Int2FloatMethod"].Request().Struct().FieldById(1).Type()
}

func getExampleJSONStringDesc() *thrift.TypeDescriptor {
	opts := thrift.Options{}
	svc, err := opts.NewDescritorFromPath(context.Background(), exampleIDLPath)
	if err != nil {
		panic(err)
	}
	return svc.Functions()["JSONStringMethod"].Request().Struct().FieldById(1).Type()
}

func getExampleFallbackDesc() *thrift.TypeDescriptor {
	opts := thrift.Options{}
	svc, err := opts.NewDescritorFromPath(context.Background(), exampleIDLPath)
	if err != nil {
		panic(err)
	}
	return svc.Functions()["FallbackMethod"].Request().Struct().FieldById(1).Type()
}

func getExampleDescByName(method string, req bool, opts thrift.Options) *thrift.TypeDescriptor {
	svc, err := opts.NewDescritorFromPath(context.Background(), exampleIDLPath)
	if err != nil {
		panic(err)
	}
	if req {
		return svc.Functions()[method].Request().Struct().Fields()[0].Type()
	} else {
		return svc.Functions()[method].Response().Struct().Fields()[0].Type()
	}
}

func getExampleData() []byte {
	out, err := ioutil.ReadFile(exampleJSON)
	if err != nil {
		panic(err)
	}
	return out
}

func getExampleReq(exp *example3.ExampleReq, setIs bool, body []byte) *http.HTTPRequest {
	f := -1.00001
	x := true
	q := []string{"1", "2", "3"}
	p := "<>"
	is := "abcd"
	uv := url.Values{
		"query": []string{strings.Join(q, ",")},
	}
	if setIs {
		uv.Add("inner_query", is)
		exp.InnerBase.InnerQuery = is
		exp.InnerBase.ListInnerBase[0].InnerQuery = is
		exp.InnerBase.MapStringInnerBase["innerx"].InnerQuery = is
	}

	uri := "http://localhost:8888/root?" + uv.Encode()
	hr, err := stdh.NewRequest("POST", uri, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	hr.Header.Set("Content-Type", "application/json")
	req, err := http.NewHTTPRequestFromStdReq(hr)
	if err != nil {
		panic(err)
	}
	req.Params.Set("path", p)
	req.Request.Header.Add("heeader", strconv.FormatBool(x))
	req.AddCookie(&stdh.Cookie{Name: "cookie", Value: strconv.FormatFloat(f, 'f', -1, 64)})
	if setIs {
		req.Request.Header.Add("inner_string", is)
		exp.InnerBase.ListInnerBase[0].String_ = is
		exp.InnerBase.MapStringInnerBase["innerx"].String_ = is
		exp.InnerBase.String_ = is
	}
	exp.Path = p
	exp.Query = q
	exp.Header = &x
	exp.Cookie = &f
	exp.RawUri = uri
	return req
}

func getExampleJSONStringReq(exp *example3.ExampleJSONString) *http.HTTPRequest {
	j := `{"a":"1","b":2}`
	x := "{}"
	a := `["1","2","3"]`
	b := `[1,2,3]`
	c := `{"1":"1","2":"2","3":"3"}`

	qs := url.Values{}
	qs.Add("query", j)
	qs.Add("query2", a)
	hr, err := stdh.NewRequest("POST", "http://localhost:8888/root?"+qs.Encode(), bytes.NewBuffer(nil))
	if err != nil {
		panic(err)
	}
	req := &http.HTTPRequest{
		Request: hr,
	}
	req.AddCookie(&stdh.Cookie{Name: "cookie", Value: x})
	req.AddCookie(&stdh.Cookie{Name: "cookie2", Value: b})
	req.Request.Header.Set("header", j)
	req.Request.Header.Set("header2", c)

	_ = json.Unmarshal([]byte(j), &exp.Query)
	_ = json.Unmarshal([]byte(a), &exp.Query2)
	_ = json.Unmarshal([]byte(x), &exp.Cookie)
	_ = json.Unmarshal([]byte(b), &exp.Cookie2)
	_ = json.Unmarshal([]byte(j), &exp.Header)
	_ = json.Unmarshal([]byte(c), &exp.Header2)
	return req
}

func getNullDesc() *thrift.TypeDescriptor {
	opts := thrift.Options{}
	svc, err := opts.NewDescritorFromPath(context.Background(), nullIDLPath)
	if err != nil {
		panic(err)
	}
	return svc.Functions()["NullTest"].Request().Struct().FieldById(1).Type()
}

func getNullData() []byte {
	out, err := ioutil.ReadFile(nullJSON)
	if err != nil {
		panic(err)
	}
	return out
}

func getNullErrData() []byte {
	out, err := ioutil.ReadFile(nullerrJSON)
	if err != nil {
		panic(err)
	}
	return out
}

func TestWriteDefault(t *testing.T) {
	desc := getExampleDesc()
	data := []byte(`{"Path":"<>"}`)
	exp := example3.NewExampleReq()
	exp.InnerBase = &example3.InnerBase{}
	data2 := []byte(`{"Path":"<>","Base":{}}`)
	err := json.Unmarshal(data2, exp)
	require.Nil(t, err)
	req := getExampleReq(exp, false, data)
	cv := NewBinaryConv(conv.Options{
		WriteDefaultField: true,
		EnableHttpMapping: true,
	})
	ctx := context.Background()
	ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
	out, err := cv.Do(ctx, desc, data)
	require.Nil(t, err)
	act := example3.NewExampleReq()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	require.Equal(t, exp, act)
}

func TestWriteRequired(t *testing.T) {
	desc := getExampleDesc()
	data := []byte(`{}`)
	t.Run("JSON", func(t *testing.T) {
		exp := example3.NewExampleReq()
		data2 := []byte(`{"Path":""}`)
		err := json.Unmarshal(data2, exp)
		require.Nil(t, err)
		cv := NewBinaryConv(conv.Options{
			WriteRequireField: true,
		})
		ctx := context.Background()
		out, err := cv.Do(ctx, desc, data)
		require.Nil(t, err)
		act := example3.NewExampleReq()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
	t.Run("http-mapping", func(t *testing.T) {
		exp := example3.NewExampleReq()
		exp.InnerBase = &example3.InnerBase{}
		data2 := []byte(`{"Path":"","Base":{}}`)
		err := json.Unmarshal(data2, exp)
		require.Nil(t, err)
		hr, err := stdh.NewRequest("POST", "http://localhost:8888/root", bytes.NewBuffer(data))
		require.NoError(t, err)
		exp.RawUri = hr.URL.String()
		req, err := http.NewHTTPRequestFromStdReq(hr)
		require.NoError(t, err)
		cv := NewBinaryConv(conv.Options{
			WriteRequireField: true,
			WriteDefaultField: true,
			EnableHttpMapping: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, data)
		require.Nil(t, err)
		act := example3.NewExampleReq()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
}

func TestBodyFallbackToHttp(t *testing.T) {
	desc := getExampleDesc()
	data := []byte(`{}`)
	hr, err := stdh.NewRequest("POST", "http://localhost:8888/root?Msg=a&Subfix=1", bytes.NewBuffer(nil))
	hr.Header.Set("Base", `{"LogID":"c"}`)
	hr.Header.Set("Path", `b`)
	// NOTICE: optional field will be ignored
	hr.Header.Set("Extra", `{"x":"y"}`)
	require.NoError(t, err)
	req := &http.HTTPRequest{
		Request: hr,
		BodyMap: map[string]string{
			"InnerBase": `{"Bool":true}`,
		},
	}

	t.Run("write default", func(t *testing.T) {
		edata := []byte(`{"Base":{"LogID":"c"},"Subfix":1,"Path":"b","InnerBase":{"Bool":true}}`)
		exp := example3.NewExampleReq()
		exp.InnerBase = sample.GetEmptyInnerBase3()
		exp.RawUri = req.GetUri()
		err = json.Unmarshal(edata, exp)
		require.Nil(t, err)
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping:            true,
			WriteDefaultField:            true,
			ReadHttpValueFallback:        true,
			TracebackRequredOrRootFields: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		act := example3.NewExampleReq()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})

	t.Run("not write default", func(t *testing.T) {
		edata := []byte(`{"Base":{"LogID":"c"},"Subfix":1,"Path":"b","InnerBase":{"Bool":true}}`)
		exp := example3.NewExampleReq()
		exp.RawUri = req.GetUri()
		err = json.Unmarshal(edata, exp)
		require.Nil(t, err)
		cv := NewBinaryConv(conv.Options{
			WriteRequireField:            true,
			EnableHttpMapping:            true,
			WriteDefaultField:            false,
			ReadHttpValueFallback:        true,
			TracebackRequredOrRootFields: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		act := example3.NewExampleReq()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
}

func TestRequireness(t *testing.T) {
	desc := getErrorExampleDesc()
	data := []byte(`{}`)
	cv := NewBinaryConv(conv.Options{
		EnableHttpMapping: true,
	})
	ctx := context.Background()
	req := http.NewHTTPRequest()
	req.Request, _ = stdh.NewRequest("GET", "root?query=abc", bytes.NewBuffer(nil))
	ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
	out, err := cv.Do(ctx, desc, data)
	require.Nil(t, err)
	act := example3.NewExampleError()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	require.Equal(t, req.URL.Query().Get("query"), act.Query)
}

func TestNullJSON2Thrift(t *testing.T) {
	desc := getNullDesc()
	data := getNullData()
	cv := NewBinaryConv(conv.Options{})
	ctx := context.Background()
	out, err := cv.Do(ctx, desc, data)
	require.Nil(t, err)
	exp := null.NewNullStruct()
	err = json.Unmarshal(data, exp)
	require.Nil(t, err)

	var m = map[string]interface{}{}
	err = json.Unmarshal(data, &m)
	require.Nil(t, err)
	fmt.Printf("%#v", m)

	act := null.NewNullStruct()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	// require.Equal(t, exp, act)
}

func TestApiBody(t *testing.T) {
	t.Run("http", func(t *testing.T) {
		desc := getExampleDescByName("ApiBodyMethod", true, thrift.Options{
			ApiBodyFastPath: true,
		})
		data := []byte(`{"code":1024,"Code":2048,"InnerCode":{}}`)
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping:            true,
			WriteDefaultField:            true,
			ReadHttpValueFallback:        true,
			TracebackRequredOrRootFields: true,
		})
		ctx := context.Background()
		req, err := stdh.NewRequest("POST", "http://localhost:8888/root", bytes.NewBuffer(data))
		require.Nil(t, err)
		req.Header.Set("Content-Type", "application/json")
		r, _ := http.NewHTTPRequestFromStdReq(req)
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, r)
		out, err := cv.Do(ctx, desc, data)
		require.Nil(t, err)
		act := example3.NewExampleApiBody()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, int64(1024), act.Code)
		require.Equal(t, int16(1024), act.Code2)
		require.Equal(t, int64(1024), act.InnerCode.C1)
		require.Equal(t, int16(0), act.InnerCode.C2)
	})
	t.Run("not http", func(t *testing.T) {
		desc := getExampleDescByName("ApiBodyMethod", true, thrift.Options{
			ApiBodyFastPath: false,
		})
		data := []byte(`{"code":1024,"Code":2048,"InnerCode":{"C1":1,"code":2}}`)
		cv := NewBinaryConv(conv.Options{
			WriteDefaultField: true,
		})
		out, err := cv.Do(context.Background(), desc, data)
		require.Nil(t, err)
		act := example3.NewExampleApiBody()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, int64(2048), act.Code)
		require.Equal(t, int16(1024), act.Code2)
		require.Equal(t, int64(1), act.InnerCode.C1)
		require.Equal(t, int16(2), act.InnerCode.C2)
	})
}

func TestFloat2Int(t *testing.T) {
	t.Run("double2int", func(t *testing.T) {
		desc := getExampleInt2FloatDesc()
		data := []byte(`{"Int32":2.229e+2}`)
		cv := NewBinaryConv(conv.Options{})
		ctx := context.Background()
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		exp := example3.NewExampleInt2Float()
		_, err = exp.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp.Int32, int32(222))
	})
	t.Run("int2double", func(t *testing.T) {
		desc := getExampleInt2FloatDesc()
		data := []byte(`{"Float64":` + strconv.Itoa(math.MaxInt64) + `}`)
		cv := NewBinaryConv(conv.Options{})
		ctx := context.Background()
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		exp := example3.NewExampleInt2Float()
		_, err = exp.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp.Float64, float64(math.MaxInt64))
	})
}

func TestEmptyConvHTTP2Thrift(t *testing.T) {
	desc := getExampleDesc()
	data := []byte(``)
	exp := example3.NewExampleReq()
	req := getExampleReq(exp, false, data)
	cv := NewBinaryConv(conv.Options{
		EnableHttpMapping: true,
	})
	ctx := context.Background()
	ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
	out, err := cv.Do(ctx, desc, data)
	require.NoError(t, err)

	act := example3.NewExampleReq()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	require.Equal(t, exp, act)
}

func TestThriftRequestBase(t *testing.T) {
	desc := getExampleDescByName("ExampleMethod", true, thrift.Options{
		// NOTICE: must set options.EnableThriftBase to true
		EnableThriftBase: true,
	})
	cv := NewBinaryConv(conv.Options{
		EnableThriftBase:  true,
		WriteDefaultField: true,
	})
	ctx := context.Background()
	b := base.NewBase()
	b.Caller = "caller"
	b.Extra = map[string]string{
		"key": "value",
	}
	ctx = context.WithValue(ctx, conv.CtxKeyThriftReqBase, b)
	app, err := json.Marshal(b)
	require.NoError(t, err)
	data := getExampleData()

	t.Run("context base", func(t *testing.T) {
		root, _ := sjson.NewSearcher(string(data)).GetByPath()
		_, err := root.Unset("Base")
		require.NoError(t, err)
		str, _ := root.Raw()
		in := []byte(str)
		out, err := cv.Do(ctx, desc, in)
		require.NoError(t, err)
		act := example3.NewExampleReq()
		_, err = act.FastRead(out)

		exp := example3.NewExampleReq()
		_, err = root.Set("Base", sjson.NewRaw(string(app)))
		require.NoError(t, err)
		str, _ = root.Raw()
		err = json.Unmarshal([]byte(str), exp)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})

	// NOTICE: when both body and context base are set, body base will be used
	t.Run("ctx + json base", func(t *testing.T) {
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		act := example3.NewExampleReq()
		_, err = act.FastRead(out)
		exp := example3.NewExampleReq()
		err = json.Unmarshal(data, exp)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
}

func TestString2Int(t *testing.T) {
	desc := getExampleInt2FloatDesc()
	cv := NewBinaryConv(conv.Options{
		String2Int64: true,
	})
	t.Run("converting", func(t *testing.T) {
		data := []byte(`{"Int32":"", "Float64":"1.1", "中文": 123.3}`)
		cv.SetOptions(conv.Options{
			EnableValueMapping: true,
		})
		ctx := context.Background()
		out, err := cv.Do(ctx, desc, data)
		require.Nil(t, err)
		exp := example3.NewExampleInt2Float()
		exp.Int32 = 0
		exp.Float64 = 1.1
		exp.String_ = "123.3"
		act := example3.NewExampleInt2Float()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})

	t.Run("no-converting", func(t *testing.T) {
		data := []byte(`{"Int32":222, "Float64":1.1, "中文": "123.3"}`)
		cv.SetOptions(conv.Options{
			EnableValueMapping: true,
		})
		ctx := context.Background()
		out, err := cv.Do(ctx, desc, data)
		require.Nil(t, err)
		exp := example3.NewExampleInt2Float()
		exp.Int32 = 222
		exp.Float64 = 1.1
		exp.String_ = "123.3"
		act := example3.NewExampleInt2Float()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})

	t.Run("option Int64AsString", func(t *testing.T) {
		data := []byte(`{"Int32":"222","Int64":"333", "Float64":"1.1"}`)
		cv.SetOptions(conv.Options{
			String2Int64: true,
		})
		ctx := context.Background()
		out, err := cv.Do(ctx, desc, data)
		require.Nil(t, err)
		exp := example3.NewExampleInt2Float()
		exp.Int64 = 333
		exp.Int32 = 222
		exp.Float64 = 1.1
		act := example3.NewExampleInt2Float()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
}

func TestJSONString(t *testing.T) {
	desc := getExampleJSONStringDesc()
	data := []byte(``)
	exp := example3.NewExampleJSONString()
	req := getExampleJSONStringReq(exp)
	cv := NewBinaryConv(conv.Options{
		EnableHttpMapping: true,
	})
	ctx := context.Background()
	ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
	out, err := cv.Do(ctx, desc, data)
	require.NoError(t, err)

	act := example3.NewExampleJSONString()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	require.Equal(t, exp, act)
}

func TestHttpConvError(t *testing.T) {
	desc := getErrorExampleDesc()
	t.Run("nil required", func(t *testing.T) {
		data := []byte(`{}`)
		hr, err := stdh.NewRequest("GET", "http://localhost", nil)
		require.Nil(t, err)
		req := &http.HTTPRequest{
			Request: hr,
		}
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		_, err = cv.Do(ctx, desc, data)
		require.Error(t, err)
		require.Equal(t, meta.ErrMissRequiredField, err.(meta.Error).Code.Behavior())
	})

	t.Run("write default", func(t *testing.T) {
		data := []byte(`{}`)
		hr, err := stdh.NewRequest("GET", "http://localhost?query=a", nil)
		require.Nil(t, err)
		hr.Header = stdh.Header{}
		req := &http.HTTPRequest{
			Request: hr,
		}
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping: true,
			WriteDefaultField: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		var exp = example3.NewExampleError()
		_, err = exp.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, "a", exp.Query)
		require.Equal(t, "", exp.Header)
	})

	t.Run("dismathed type", func(t *testing.T) {
		data := []byte(`{}`)
		hr, err := stdh.NewRequest("GET", "http://localhost?query=a&q2=1.5", nil)
		require.Nil(t, err)
		req := &http.HTTPRequest{
			Request: hr,
		}
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		_, err = cv.Do(ctx, desc, data)
		require.Error(t, err)
		require.Equal(t, meta.ErrConvert, err.(meta.Error).Code.Behavior())
	})
}

func TestHttpMappingFallback(t *testing.T) {
	desc := getExampleFallbackDesc()
	data := []byte(`{"Msg":"hello","Heeader":"world"}`)
	t.Run("fallback", func(t *testing.T) {
		hr, err := stdh.NewRequest("GET", "http://localhost?query=a", nil)
		req := &http.HTTPRequest{
			Request: hr,
		}
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping:     true,
			ReadHttpValueFallback: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		exp := example3.NewExampleFallback()
		exp.Msg = "hello"
		exp.Heeader = "world"
		act := example3.NewExampleFallback()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
	t.Run("not fallback", func(t *testing.T) {
		hr, err := stdh.NewRequest("GET", "http://localhost?A=a", nil)
		hr.Header.Set("heeader", "中文")
		req := &http.HTTPRequest{
			Request: hr,
		}
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping: true,
		})
		ctx := context.Background()
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, data)
		require.NoError(t, err)
		exp := example3.NewExampleFallback()
		exp.Msg = "a"
		exp.Heeader = "中文"
		act := example3.NewExampleFallback()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
}

func TestPostFormBody(t *testing.T) {
	desc := getExampleDescByName("PostFormMethod", true, thrift.Options{})
	data := url.Values{
		"form":       []string{"b"},
		"JSON":       []string{`{"a":"中文","b":1}`},
		"inner_form": []string{"1"},
	}
	t.Run("fallback", func(t *testing.T) {
		exp := example3.NewExamplePostForm()
		exp.Query = "a"
		exp.Form = "b"
		exp.JSON = &example3.InnerJSON{
			A:         "中文",
			B:         1,
			InnerForm: 0,
		}
		cv := NewBinaryConv(conv.Options{
			WriteDefaultField:            true,
			EnableHttpMapping:            true,
			ReadHttpValueFallback:        true,
			TracebackRequredOrRootFields: true,
		})
		ctx := context.Background()
		sr, err := stdh.NewRequest("POST", "http://localhost?query=a", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		sr.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req, err := http.NewHTTPRequestFromStdReq(sr)
		require.NoError(t, err)
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, []byte(`{}`))
		require.Nil(t, err)
		act := example3.NewExamplePostForm()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
	t.Run("no fallback", func(t *testing.T) {
		exp := example3.NewExamplePostForm()
		exp.Query = "a"
		exp.Form = "b"
		// exp.JSON = &example3.InnerJSON{
		// 	A: "中文",
		// 	B: 1,
		// 	InnerForm: 1,
		// }   //NOTICE: not set since conv data is nil, thus no fallback
		cv := NewBinaryConv(conv.Options{
			WriteDefaultField: false,
			EnableHttpMapping: true,
		})
		ctx := context.Background()
		sr, err := stdh.NewRequest("POST", "http://localhost?query=a", strings.NewReader(data.Encode()))
		require.NoError(t, err)
		sr.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req, err := http.NewHTTPRequestFromStdReq(sr)
		require.NoError(t, err)
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, []byte(``))
		require.Nil(t, err)
		act := example3.NewExamplePostForm()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
}

func TestAGWDynamicBody(t *testing.T) {
	desc := getExampleDescByName("DynamicStructMethod", true, thrift.Options{})
	exp := example3.NewExampleDynamicStruct()
	exp.Query = "1"
	exp.JSON = "[1,2,3]"
	exp.InnerStruct = &example3.InnerStruct{
		InnerJSON: `{"a":"中文","b":1}`,
		Must:      "2",
	}
	t.Run("no http-mapping", func(t *testing.T) {
		data := `{"Query":"1","json":[1,2,3],"inner_struct":{"inner_json":{"a":"中文","b":1},"Must":"2"}}`
		cv := NewBinaryConv(conv.Options{
			EnableValueMapping:    true,
			EnableHttpMapping:     false,
			WriteRequireField:     true,
			ReadHttpValueFallback: true,
		})
		ctx := context.Background()
		out, err := cv.Do(ctx, desc, []byte(data))
		require.Nil(t, err)
		act := example3.NewExampleDynamicStruct()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
	t.Run("http-mapping", func(t *testing.T) {
		data := `{"json":[1,2,3],"inner_struct":{"inner_json":{"a":"中文","b":1}}}`
		cv := NewBinaryConv(conv.Options{
			EnableValueMapping:           true,
			EnableHttpMapping:            true,
			WriteRequireField:            true,
			ReadHttpValueFallback:        true,
			TracebackRequredOrRootFields: true,
		})
		ctx := context.Background()
		req, err := stdh.NewRequest("GET", "http://localhost?query=1&Must=2", nil)
		require.NoError(t, err)
		rr, err := http.NewHTTPRequestFromStdReq(req)
		require.NoError(t, err)
		ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, rr)
		out, err := cv.Do(ctx, desc, []byte(data))
		require.Nil(t, err)
		act := example3.NewExampleDynamicStruct()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, exp, act)
	})
}

func TestNobodyRequiredFields(t *testing.T) {
	path := "a/b/main.thrift"
	content := `
	namespace go kitex.test.server
	struct Base {
		1: required string required_field
	}

	service InboxService {
		string ExampleMethod(1: Base req)
	}
	`
	includes := map[string]string{
		path: content,
	}
	p, err := thrift.NewDescritorFromContent(context.Background(), path, content, includes, true)
	if err != nil {
		t.Fatal(err)
	}
	desc := p.Functions()["ExampleMethod"].Request().Struct().Fields()[0].Type()
	cv := NewBinaryConv(conv.Options{
		EnableHttpMapping:     true,
		WriteRequireField:     true,
		ReadHttpValueFallback: true,
	})
	ctx := context.Background()
	req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost?required_field=1", nil)
	require.NoError(t, err)
	ctx = context.WithValue(ctx, conv.CtxKeyHTTPRequest, req)
	out, err := cv.Do(ctx, desc, nil)
	require.Nil(t, err)
	fmt.Printf("%+v", out)
}

func TestBase64Decode(t *testing.T) {
	desc := getExampleDescByName("Base64BinaryMethod", true, thrift.Options{})
	t.Run("base64 decode", func(t *testing.T) {
		exp := example3.NewExampleBase64Binary()
		exp.Binary = []byte("hello")
		in, err := json.Marshal(exp)
		require.Nil(t, err)
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping: true,
			NoBase64Binary:    false,
		})
		req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost", nil)
		require.NoError(t, err)
		req.Request.Header.Set("Binary2", base64.StdEncoding.EncodeToString([]byte("world")))
		ctx := context.WithValue(context.Background(), conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, in)
		require.Nil(t, err)

		act := example3.NewExampleBase64Binary()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, []byte("hello"), act.Binary)
		require.Equal(t, []byte("world"), act.Binary2)
	})

	t.Run("no base64 decode", func(t *testing.T) {
		in := []byte(`{"Binary":"hello"}`)
		cv := NewBinaryConv(conv.Options{
			EnableHttpMapping: true,
			NoBase64Binary:    true,
		})
		req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost", nil)
		require.NoError(t, err)
		req.Request.Header.Set("Binary2", "world")
		ctx := context.WithValue(context.Background(), conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, in)
		require.Nil(t, err)

		act := example3.NewExampleBase64Binary()
		_, err = act.FastRead(out)
		require.Nil(t, err)
		require.Equal(t, []byte("hello"), act.Binary)
		require.Equal(t, []byte("world"), act.Binary2)
	})
}

func TestDefaultValue(t *testing.T) {
	desc := getExampleDescByName("DefaultValueMethod", true, thrift.Options{
		UseDefaultValue: true,
	})
	in := []byte(`{}`)
	t.Run("default value", func(t *testing.T) {
		cv := NewBinaryConv(conv.Options{
			WriteDefaultField: true,
			EnableHttpMapping: false,
		})
		out, err := cv.Do(context.Background(), desc, in)
		require.Nil(t, err)
		act := &example3.ExampleDefaultValue{}
		_, err = act.FastRead(out)
		require.Nil(t, err)
		exp := example3.NewExampleDefaultValue()
		require.Equal(t, exp, act)
	})
	t.Run("default value + http mapping", func(t *testing.T) {
		cv := NewBinaryConv(conv.Options{
			WriteDefaultField: true,
			EnableHttpMapping: true,
		})
		req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost", nil)
		require.NoError(t, err)
		ctx := context.WithValue(context.Background(), conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, in)
		require.Nil(t, err)
		act := &example3.ExampleDefaultValue{}
		_, err = act.FastRead(out)
		require.Nil(t, err)

		exp := example3.NewExampleDefaultValue()
		require.Equal(t, exp, act)
	})
	t.Run("zero value", func(t *testing.T) {
		desc := getExampleDescByName("DefaultValueMethod", true, thrift.Options{
			UseDefaultValue: false,
		})
		cv := NewBinaryConv(conv.Options{
			WriteDefaultField: true,
			EnableHttpMapping: false,
		})
		req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost", nil)
		require.NoError(t, err)
		ctx := context.WithValue(context.Background(), conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, in)
		require.Nil(t, err)
		act := &example3.ExampleDefaultValue{}
		_, err = act.FastRead(out)
		require.Nil(t, err)

		exp := &example3.ExampleDefaultValue{}
		require.Equal(t, exp, act)
	})
}

func TestOptionalDefaultValue(t *testing.T) {
	desc := getExampleDescByName("OptionalDefaultValueMethod", true, thrift.Options{
		SetOptionalBitmap: true,
		UseDefaultValue:   true,
	})
	in := []byte(`{}`)
	t.Run("write default + write optional", func(t *testing.T) {
		cv := NewBinaryConv(conv.Options{
			WriteRequireField:  true,
			WriteDefaultField:  true,
			EnableHttpMapping:  false,
			WriteOptionalField: true,
		})
		out, err := cv.Do(context.Background(), desc, in)
		require.Nil(t, err)
		act := &example3.ExampleOptionalDefaultValue{}
		_, err = act.FastRead(out)
		require.Nil(t, err)
		exp := example3.NewExampleOptionalDefaultValue()
		exp.E = new(string)
		exp.F = new(string)
		require.Equal(t, exp, act)
	})
	t.Run("not write optional", func(t *testing.T) {
		cv := NewBinaryConv(conv.Options{
			WriteRequireField: true,
			WriteDefaultField: false,
			EnableHttpMapping: false,
		})
		out, err := cv.Do(context.Background(), desc, in)
		require.Nil(t, err)
		act := &example3.ExampleOptionalDefaultValue{}
		_, err = act.FastRead(out)
		require.Nil(t, err)
		exp := example3.NewExampleOptionalDefaultValue()
		exp.A = ""
		exp.C = 0
		require.Equal(t, exp, act)
	})
	t.Run("write default", func(t *testing.T) {
		cv := NewBinaryConv(conv.Options{
			WriteRequireField: false,
			WriteDefaultField: true,
			EnableHttpMapping: false,
		})
		_, err := cv.Do(context.Background(), desc, in)
		require.Error(t, err)
	})
	t.Run("write default + http mapping", func(t *testing.T) {
		in := []byte(`{"B":1}`)
		cv := NewBinaryConv(conv.Options{
			WriteRequireField: false,
			WriteDefaultField: true,
			EnableHttpMapping: true,
		})
		req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost", nil)
		require.NoError(t, err)
		ctx := context.WithValue(context.Background(), conv.CtxKeyHTTPRequest, req)
		_, err = cv.Do(ctx, desc, in)
		require.Error(t, err)
	})
	t.Run("write default + write optional + http mapping", func(t *testing.T) {
		cv := NewBinaryConv(conv.Options{
			WriteRequireField:  true,
			WriteDefaultField:  true,
			EnableHttpMapping:  true,
			WriteOptionalField: true,
		})
		req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost", nil)
		require.NoError(t, err)
		ctx := context.WithValue(context.Background(), conv.CtxKeyHTTPRequest, req)
		out, err := cv.Do(ctx, desc, in)
		require.Nil(t, err)
		act := &example3.ExampleOptionalDefaultValue{}
		_, err = act.FastRead(out)
		require.Nil(t, err)
		exp := example3.NewExampleOptionalDefaultValue()
		exp.E = new(string)
		exp.F = new(string)
		require.Equal(t, exp, act)
	})
}

func TestNoBodyStruct(t *testing.T) {
	desc := getExampleDescByName("NoBodyStructMethod", true, thrift.Options{
		UseDefaultValue: true,
	})
	in := []byte(`{}`)
	req, err := http.NewHTTPRequestFromUrl("GET", "http://localhost?b=1", nil)
	require.NoError(t, err)
	cv := NewBinaryConv(conv.Options{
		EnableHttpMapping: true,
	})
	ctx := context.WithValue(context.Background(), conv.CtxKeyHTTPRequest, req)
	out, err := cv.Do(ctx, desc, in)
	require.Nil(t, err)
	act := &example3.ExampleNoBodyStruct{}
	_, err = act.FastRead(out)
	require.Nil(t, err)
	exp := example3.NewExampleNoBodyStruct()
	exp.NoBodyStruct = example3.NewNoBodyStruct()
	B := int32(1)
	exp.NoBodyStruct.B = &B
	require.Equal(t, exp, act)
}

func TestSimpleArgs(t *testing.T) {
	cv := NewBinaryConv(conv.Options{})

	t.Run("string", func(t *testing.T) {
		desc := getExampleDescByName("String", true, thrift.Options{})
		p := thrift.NewBinaryProtocolBuffer()
		p.WriteString("hello")
		exp := p.Buf
		out, err := cv.Do(context.Background(), desc, []byte(`"hello"`))
		require.NoError(t, err)
		require.Equal(t, exp, out)
	})

	t.Run("no quoted string", func(t *testing.T) {
		desc := getExampleDescByName("String", true, thrift.Options{})
		p := thrift.NewBinaryProtocolBuffer()
		p.WriteString(`hel\lo`)
		exp := p.Buf
		out, err := cv.Do(context.Background(), desc, []byte(`hel\lo`))
		require.NoError(t, err)
		require.Equal(t, exp, out)
	})

	t.Run("int", func(t *testing.T) {
		desc := getExampleDescByName("I64", true, thrift.Options{})
		p := thrift.NewBinaryProtocolBuffer()
		p.WriteI64(math.MaxInt64)
		exp := p.Buf
		out, err := cv.Do(context.Background(), desc, []byte(strconv.Itoa(math.MaxInt64)))
		require.NoError(t, err)
		require.Equal(t, exp, out)
	})
}

func TestEmptyStruct(t *testing.T) {
	desc, err := thrift.NewDescritorFromContent(context.Background(), "a/b/main.thrift", `
struct Req {}

service Svc {
	void Method(1: Req req)
}
	`, nil, true)
	require.NoError(t, err)
	reqDesc := desc.Functions()["Method"].Request().Struct().FieldById(1).Type()
	cv := NewBinaryConv(conv.Options{})
	out, err := cv.Do(context.Background(), reqDesc, []byte(`{"UNKNOWN":1}`))
	require.NoError(t, err)
	fmt.Printf("%+v", out)
}

func TestMergeBase(t *testing.T) {
	opts := thrift.Options{
		EnableThriftBase: true,
	}
	svc, err := opts.NewDescritorFromPath(context.Background(), exampleIDLPath)
	if err != nil {
		panic(err)
	}
	desc := svc.Functions()["ExampleMethod"].Request().Struct().FieldById(1).Type()
	data := getExampleData()
	cv := NewBinaryConv(conv.Options{
		EnableThriftBase: true,
		MergeBaseFunc:    mergeBase,
	})
	root := ast.NewRaw(string(data))
	root.Set("Base", ast.NewRaw(`{"LogID":"2","Client":"2","Extra":{"a":"2","c":"2"}}`))
	js, err := root.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	data = js
	ctx := context.Background()
	ctx = context.WithValue(ctx, conv.CtxKeyThriftReqBase, &base.Base{
		LogID: "1",
		Extra: map[string]string{"a": "1", "b": "1"},
	})
	out, err := cv.Do(ctx, desc, data)
	require.Nil(t, err)
	exp := example3.NewExampleReq()
	err = json.Unmarshal(data, exp)
	require.Nil(t, err)
	exp.Base = &ebase.Base{
		LogID:  "1",
		Client: "",
		Extra:  map[string]string{"a": "1", "b": "1", "c": "2"},
	}
	act := example3.NewExampleReq()
	_, err = act.FastRead(out)
	require.Nil(t, err)
	require.Equal(t, exp, act)
}

// MergeBase merge `to` into `from`
func mergeBase(from base.Base, to base.Base) base.Base {
	from.LogID = to.LogID
	from.Caller = to.Caller
	from.Addr = to.Addr
	from.Client = to.Client
	from.TrafficEnv = to.TrafficEnv
	if to.Extra != nil {
		if from.Extra == nil {
			from.Extra = to.Extra
		} else {
			for k, v := range to.Extra {
				from.Extra[k] = v
			}
		}
	}
	return from
}
