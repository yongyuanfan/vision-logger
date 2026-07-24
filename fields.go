package visionlogger

// 核心字段名（与 slog JSONHandler / Filebeat ndjson 契约对齐，业务不得改名）。
const (
	FieldTime  = "time"
	FieldLevel = "level"
	FieldMsg   = "msg"
)

// 推荐扩展字段名（扁平 attrs，由业务按需追加）。
const (
	FieldComponent      = "component"
	FieldTraceID        = "trace_id"
	FieldUserID         = "user_id"
	FieldRunID          = "run_id"
	FieldConversationID = "conversation_id"
	// FieldService 表示传输面（如 http/grpc）；Filebeat 会将其 rename 为 transport。
	FieldService = "service"
)
