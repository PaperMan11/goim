package a2r

import (
	"context"
	"reflect"

	"github.com/PaperMan11/goim/pkg/apiresp"
	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logc"
	"google.golang.org/grpc"
)

func Call[A, B any](c *gin.Context, rpc func(ctx context.Context, req A, opts ...grpc.CallOption) (B, error)) {
	var req A
	allocIfNilPtr(&req)
	if err := c.ShouldBind(&req); err != nil {
		writeErr(c, err)
		return
	}

	resp, err := rpc(c, req)
	respond(c, resp, err)
}

// CallQuery 用于 GET 类接口：先把 query 参数绑定到带 form tag 的 Q，
// 再通过 build 转换成 protobuf 请求调用 RPC。
// protobuf 生成的 struct 只有 json tag，直接绑定 query 会因字段名大小写不匹配而取不到值。
func CallQuery[A, Q, B any](c *gin.Context, build func(Q) A, rpc func(ctx context.Context, req A, opts ...grpc.CallOption) (B, error)) {
	var query Q
	if err := c.ShouldBindQuery(&query); err != nil {
		writeErr(c, err)
		return
	}

	resp, err := rpc(c, build(query))
	respond(c, resp, err)
}

func respond[B any](c *gin.Context, resp B, err error) {
	if err != nil {
		writeErr(c, err)
		return
	}
	apiresp.Success(c.Writer, resp)
}

func writeErr(c *gin.Context, err error) {
	logc.Errorf(c.Request.Context(), "A2R failed: %v, %s %s", err, c.Request.Method, c.Request.URL.Path)
	apiresp.Error(c.Writer, err)
}

// allocIfNilPtr 为指针类型的请求分配实例。
// A 通常是 protobuf 生成的 *pb.XxxReq，零值为 nil；gin v1.12 的默认校验器
// 遇到指向 nil 指针的请求会执行 reflect.Value.Elem().Interface() 而 panic，
// 因此绑定前先把 nil 指针初始化掉。
func allocIfNilPtr(req any) {
	rv := reflect.ValueOf(req)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return
	}
	elem := rv.Elem()
	if elem.Kind() == reflect.Ptr && elem.IsNil() {
		elem.Set(reflect.New(elem.Type().Elem()))
	}
}
