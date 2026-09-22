package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	cls "github.com/tencentcloud/tencentcloud-cls-sdk-go"
)

// clsCloseTimeoutMs 进程退出时等待 CLS 刷出内存批次的上限。
// 与 vision-ai 服务退出预算同量级，避免关闭被卡住。
const clsCloseTimeoutMs int64 = 5000

// clsSender 抽象 CLS 异步发送，测试可替换为假实现，避免访问腾讯云。
type clsSender interface {
	SendLog(topicID string, log *cls.Log, callback cls.CallBack) error
}

type clsCloser interface {
	Close(timeoutMs int64) error
}

type clsFailCallback struct{}

func (clsFailCallback) Success(*cls.Result) {}

// Fail 只写 stderr。若再写回本 logger，失败回调会递归打日志。
func (clsFailCallback) Fail(result *cls.Result) {
	if result == nil {
		fmt.Fprintln(os.Stderr, "cls send log failed")
		return
	}
	fmt.Fprintf(os.Stderr, "cls send log failed: code=%s message=%s request_id=%s\n",
		result.GetErrorCode(), result.GetErrorMessage(), result.GetRequestId())
}

type clsHandler struct {
	sender  clsSender
	topicID string
	level   slog.Level
	attrs   []slog.Attr
	groups  []string
}

func newCLSHandler(sender clsSender, topicID string, level slog.Level) *clsHandler {
	return &clsHandler{sender: sender, topicID: topicID, level: level}
}

func (h *clsHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *clsHandler) Handle(_ context.Context, record slog.Record) error {
	fields := make(map[string]string, len(h.attrs)+8)
	prefix := strings.Join(h.groups, ".")
	add := func(attr slog.Attr) {
		attr.Value = attr.Value.Resolve()
		if attr.Key == "" {
			return
		}
		key := attr.Key
		if prefix != "" {
			key = prefix + "." + key
		}
		fields[key] = attrString(attr.Value)
	}
	for _, attr := range h.attrs {
		add(attr)
	}
	record.Attrs(func(attr slog.Attr) bool {
		add(attr)
		return true
	})
	fields[FieldTime] = record.Time.Format(time.RFC3339Nano)
	fields[FieldLevel] = record.Level.String()
	fields[FieldMsg] = record.Message

	if err := h.sender.SendLog(h.topicID, cls.NewCLSLog(record.Time.Unix(), fields), clsFailCallback{}); err != nil {
		fmt.Fprintf(os.Stderr, "cls send log: %v\n", err)
		return err
	}
	return nil
}

func (h *clsHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	copied := *h
	copied.attrs = append(append([]slog.Attr{}, h.attrs...), attrs...)
	return &copied
}

func (h *clsHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	copied := *h
	copied.groups = append(append([]string{}, h.groups...), name)
	return &copied
}

func openCLSLocked(cfg CLSConfig) error {
	cfg = cfg.trimmed()
	if err := cfg.validate(); err != nil {
		return err
	}
	clientCfg := cls.GetDefaultAsyncProducerClientConfig()
	clientCfg.Endpoint = cfg.Endpoint
	clientCfg.AccessKeyID = cfg.AccessKeyID
	clientCfg.AccessKeySecret = cfg.AccessKeySecret
	clientCfg.AccessToken = cfg.AccessToken
	// 缓冲满时立即失败并丢弃，不阻塞业务请求。
	clientCfg.MaxBlockSec = 0

	producer, err := cls.NewAsyncProducerClient(clientCfg)
	if err != nil {
		return fmt.Errorf("create cls producer: %w", err)
	}
	producer.Start()
	logger = slog.New(newCLSHandler(producer, cfg.TopicID, minLevel))
	clsClient = producer
	return nil
}

func (c CLSConfig) trimmed() CLSConfig {
	c.Endpoint = strings.TrimSpace(c.Endpoint)
	c.TopicID = strings.TrimSpace(c.TopicID)
	c.AccessKeyID = strings.TrimSpace(c.AccessKeyID)
	c.AccessKeySecret = strings.TrimSpace(c.AccessKeySecret)
	c.AccessToken = strings.TrimSpace(c.AccessToken)
	return c
}

func (c CLSConfig) validate() error {
	if c.Endpoint == "" {
		return fmt.Errorf("cls endpoint 不能为空")
	}
	if c.TopicID == "" {
		return fmt.Errorf("cls topic_id 不能为空")
	}
	if c.AccessKeyID == "" || c.AccessKeySecret == "" {
		return fmt.Errorf("cls access_key_id 与 access_key_secret 不能为空")
	}
	return nil
}

func attrString(value slog.Value) string {
	value = value.Resolve()
	switch value.Kind() {
	case slog.KindString, slog.KindInt64, slog.KindUint64, slog.KindFloat64, slog.KindBool:
		return value.String()
	case slog.KindDuration:
		return value.Duration().String()
	case slog.KindTime:
		return value.Time().Format(time.RFC3339Nano)
	case slog.KindGroup:
		grouped := make(map[string]string, len(value.Group()))
		for _, attr := range value.Group() {
			grouped[attr.Key] = attrString(attr.Value)
		}
		encoded, err := json.Marshal(grouped)
		if err != nil {
			return value.String()
		}
		return string(encoded)
	default:
		encoded, err := json.Marshal(value.Any())
		if err != nil {
			return value.String()
		}
		return string(encoded)
	}
}

func normalizeDriver(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", DriverFile:
		return DriverFile, nil
	case DriverCLS:
		return DriverCLS, nil
	default:
		return "", fmt.Errorf("logging driver %q 无效，仅支持 file 或 cls", strings.TrimSpace(raw))
	}
}
