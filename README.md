# vision-logger

Vision 平台统一日志 SDK（Go）。各业务服务通过本包输出 **NDJSON** 到本地 `runtime/logs`，由 Filebeat 旁路采集写入 Elasticsearch；**本 SDK 不内置 ES 客户端、不发起观测上报**。

## 快速接入

```go
import visionlogger "github.com/yongyuanfan/vision-logger"

func main() {
    if err := visionlogger.Init(visionlogger.Config{
        Enabled:  true,
        Level:    "info", // debug | info | warn | error
        FilePath: "runtime/logs/vision-ai.log",
    }); err != nil {
        panic(err)
    }
    defer visionlogger.Close()

    visionlogger.Info("server starting",
        visionlogger.FieldComponent, "runtime",
        visionlogger.FieldService, "http",
    )
}
```

本地多仓并列时，业务 `go.mod` 可使用：

```go
require github.com/yongyuanfan/vision-logger v0.0.0
replace github.com/yongyuanfan/vision-logger => ../vision-logger
```

## 落盘行为

| 项 | 说明 |
|----|------|
| 格式 | 一行一条 JSON（兼容 `log/slog` JSONHandler） |
| 基路径 | 如 `runtime/logs/vision-ai.log` |
| 按天滚动 | `runtime/logs/vision-ai-2026-07-09.log` |
| 未启用 | `Enabled=false` 时全部 no-op，不创建文件 |
| 故障隔离 | 写文件失败不影响业务主路径（当前写失败静默跳过） |

## 字段契约

### 核心字段（SDK 保证）

| 字段 | 说明 |
|------|------|
| `time` | RFC3339Nano |
| `level` | `DEBUG` / `INFO` / `WARN` / `ERROR` |
| `msg` | 消息正文 |

### Filebeat → ES 映射（采集侧）

| 源 | ES 字段 |
|----|---------|
| `time` | `@timestamp` |
| `msg` | `message` |
| `level` | `level` |
| 环境变量 `SERVICE_NAME` | `service_name` |
| 环境变量 `APP_ENV` | `env` |
| 业务 attrs `service` | rename 为 `transport` |

应用名由采集环境变量注入，业务勿把应用名写进 `service`；`service` 仅表示传输面（如 `http` / `grpc`）。

### 推荐扩展 attrs

| 常量 | 字段名 | 用途 |
|------|--------|------|
| `FieldComponent` | `component` | 组件（runtime / http / grpc / mcp-tool 等） |
| `FieldTraceID` | `trace_id` | 链路追踪 |
| `FieldUserID` | `user_id` | 用户 |
| `FieldRunID` | `run_id` | 对话运行 |
| `FieldConversationID` | `conversation_id` | 会话 |
| `FieldService` | `service` | 传输面（进 ES 后为 `transport`） |

禁止写入密钥、令牌、对话正文等敏感内容。

## Context 辅助

```go
ctx := visionlogger.ContextWithTraceID(ctx, traceID)
ctx = visionlogger.ContextWithUserID(ctx, userID)
visionlogger.FromContext(ctx).Info("request handled", visionlogger.FieldComponent, "http")
```

空值不会写入 JSON。也可用 `With` 固定默认字段：

```go
log := visionlogger.With(visionlogger.FieldComponent, "runtime")
log.Info("starting")
```

## 禁止事项

- 禁止在 SDK 或业务进程内直连 Elasticsearch
- 禁止以纯文本 `log` 作为 CSS505 接入主路径
- 禁止擅自改名核心字段 `time` / `level` / `msg`
