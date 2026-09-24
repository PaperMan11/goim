package a2r

import (
	"context"

	"github.com/PaperMan11/goim/pkg/apiresp"
	"github.com/gin-gonic/gin"
	"github.com/zeromicro/go-zero/core/logc"
	"google.golang.org/grpc"
)

func Call[A, B any](c *gin.Context, rpc func(ctx context.Context, req A, opts ...grpc.CallOption) (B, error)) {
	var req A
	if err := c.ShouldBind(&req); err != nil {
		writeErr(c, err)
		return
	}

	resp, err := rpc(c.Request.Context(), req)
	if err != nil {
		writeErr(c, err)
		return
	}

	// if err := copier.Copy(&resp, &resp); err != nil {
	// 	writeErr(c, err)
	// 	return
	// }

	apiresp.Success(c.Writer, resp)
}

func writeErr(c *gin.Context, err error) {
	logc.Errorf(c.Request.Context(), "A2R failed: %v, %s %s", err, c.Request.Method, c.Request.URL.Path)
	apiresp.Error(c.Writer, err)
}
