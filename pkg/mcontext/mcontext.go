package mcontext

import (
	"context"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
)

func GetOpUserIDFromContext(ctx context.Context) string {
	return ctx.Value(constant.OpUserID).(string)
}

func GetOpUserPlatformFromContext(ctx context.Context) string {
	return ctx.Value(constant.OpUserPlatform).(string)
}

func GetConnIDFromContext(ctx context.Context) string {
	return ctx.Value(constant.ConnID).(string)
}

func GetTokenFromContext(ctx context.Context) string {
	return ctx.Value(constant.Token).(string)
}

func GetTriggerIDFromContext(ctx context.Context) string {
	return ctx.Value(constant.TriggerID).(string)
}

func GetClientIPFromContext(ctx context.Context) string {
	return ctx.Value(constant.ClientIP).(string)
}

func GetOperationIDFromContext(ctx context.Context) string {
	return ctx.Value(constant.OperationID).(string)
}

func GetRpcCustomHeaderFromContext(ctx context.Context) []string {
	return ctx.Value(constant.RpcCustomHeader).([]string)
}

func GetCheckKeyFromContext(ctx context.Context) string {
	return ctx.Value(constant.CheckKey).(string)
}

// ------------------------------------------------------------------------------------

func SetOpUserIDInContext(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, constant.OpUserID, userID)
}

func SetOpUserPlatformInContext(ctx context.Context, platformID string) context.Context {
	return context.WithValue(ctx, constant.OpUserPlatform, platformID)
}

func SetConnIDInContext(ctx context.Context, connID string) context.Context {
	return context.WithValue(ctx, constant.ConnID, connID)
}

func SetTokenInContext(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, constant.Token, token)
}

func SetTriggerIDInContext(ctx context.Context, triggerID string) context.Context {
	return context.WithValue(ctx, constant.TriggerID, triggerID)
}

func SetClientIPInContext(ctx context.Context, clientIP string) context.Context {
	return context.WithValue(ctx, constant.ClientIP, clientIP)
}

func SetOperationIDInContext(ctx context.Context, operationID string) context.Context {
	return context.WithValue(ctx, constant.OperationID, operationID)
}

func SetRpcCustomHeaderInContext(ctx context.Context, headers []string) context.Context {
	return context.WithValue(ctx, constant.RpcCustomHeader, headers)
}

func SetCheckKeyInContext(ctx context.Context, checkKey string) context.Context {
	return context.WithValue(ctx, constant.CheckKey, checkKey)
}
