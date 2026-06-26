package clientinterceptors

import (
	"context"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func ClientContextInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md, ok := metadata.FromOutgoingContext(ctx)
		if !ok {
			md = metadata.Pairs()
		}

		if keys, ok := ctx.Value(constant.RpcCustomHeader).([]string); ok {
			for _, key := range keys {
				if value, ok := ctx.Value(key).([]string); ok {
					md.Set(key, value...)
				}
			}
		}

		operationID, ok := ctx.Value(constant.OperationID).(string)
		if ok {
			md.Set(constant.OperationID, operationID)
		}
		opUserID, ok := ctx.Value(constant.OpUserID).(string)
		if ok {
			md.Set(constant.OpUserID, opUserID)
		}
		opUserPlatform, ok := ctx.Value(constant.OpUserPlatform).(string)
		if ok {
			md.Set(constant.OpUserPlatform, opUserPlatform)
		}
		token, ok := ctx.Value(constant.Token).(string)
		if ok {
			md.Set(constant.Token, token)
		}
		connID, ok := ctx.Value(constant.ConnID).(string)
		if ok {
			md.Set(constant.ConnID, connID)
		}
		clientIP, ok := ctx.Value(constant.ClientIP).(string)
		if ok {
			md.Set(constant.ClientIP, clientIP)
		}
		triggerID, ok := ctx.Value(constant.TriggerID).(string)
		if ok {
			md.Set(constant.TriggerID, triggerID)
		}

		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
