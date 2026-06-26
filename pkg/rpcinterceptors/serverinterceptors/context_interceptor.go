package serverinterceptors

import (
	"context"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func ServerContextInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "metadata is empty")
		}

		if headers := md.Get(constant.RpcCustomHeader); len(headers) > 0 {
			ctx = context.WithValue(ctx, constant.RpcCustomHeader, headers)
			for _, header := range headers {
				ctx = context.WithValue(ctx, header, md.Get(header))
			}
		}

		operationID := md.Get(constant.OperationID)
		if len(operationID) == 0 {
			return nil, status.Error(codes.InvalidArgument, "operationID is empty")
		}
		ctx = context.WithValue(ctx, constant.OperationID, operationID[0])
		if userID := md.Get(constant.OpUserID); len(userID) > 0 {
			ctx = context.WithValue(ctx, constant.OpUserID, userID[0])
		}
		if platformID := md.Get(constant.OpUserPlatform); len(platformID) > 0 {
			ctx = context.WithValue(ctx, constant.OpUserPlatform, platformID[0])
		}
		if connID := md.Get(constant.ConnID); len(connID) > 0 {
			ctx = context.WithValue(ctx, constant.ConnID, connID[0])
		}
		if token := md.Get(constant.Token); len(token) > 0 {
			ctx = context.WithValue(ctx, constant.Token, token[0])
		}
		if triggerID := md.Get(constant.TriggerID); len(triggerID) > 0 {
			ctx = context.WithValue(ctx, constant.TriggerID, triggerID[0])
		}
		if clientIP := md.Get(constant.ClientIP); len(clientIP) > 0 {
			ctx = context.WithValue(ctx, constant.ClientIP, clientIP[0])
		}

		return handler(ctx, req)
	}
}
