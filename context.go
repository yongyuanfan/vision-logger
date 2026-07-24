package visionlogger

import "context"

type ctxKey int

const (
	ctxKeyTraceID ctxKey = iota + 1
	ctxKeyUserID
)

// ContextWithTraceID 将 trace_id 写入 context，供 FromContext 自动附加。
func ContextWithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyTraceID, traceID)
}

// ContextWithUserID 将 user_id 写入 context，供 FromContext 自动附加。
func ContextWithUserID(ctx context.Context, userID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, ctxKeyUserID, userID)
}

// TraceIDFromContext 读取 context 中的 trace_id；不存在时返回空字符串。
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyTraceID).(string)
	return v
}

// UserIDFromContext 读取 context 中的 user_id；不存在时返回空字符串。
func UserIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	v, _ := ctx.Value(ctxKeyUserID).(string)
	return v
}

// FromContext 返回带有 context 中已有关联字段的 Logger；字段为空时不写入。
func FromContext(ctx context.Context) *Logger {
	attrs := make([]any, 0, 4)
	if traceID := TraceIDFromContext(ctx); traceID != "" {
		attrs = append(attrs, FieldTraceID, traceID)
	}
	if userID := UserIDFromContext(ctx); userID != "" {
		attrs = append(attrs, FieldUserID, userID)
	}
	if len(attrs) == 0 {
		return &Logger{}
	}
	return With(attrs...)
}
