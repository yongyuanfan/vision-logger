package logging

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	defaultFilePath = "runtime/logs/app.log"
	dailyDateLayout = "2006-01-02"
)

var (
	mu         sync.Mutex
	enabled    bool
	logger     *slog.Logger
	logFile    *os.File
	minLevel   slog.Level
	basePath   string
	currentDay string
	// nowFunc 便于测试按天滚动；生产环境使用 time.Now。
	nowFunc = time.Now
)

// Logger 带默认 attrs 的子日志器；零值等价于包级 Info/Warn/Debug/Error。
type Logger struct {
	attrs []any
}

// Init 根据配置初始化 JSON 文件日志；未启用时直接返回。
// 日志按天切分，实际文件名为在基路径扩展名前插入日期，例如 app-2026-07-09.log。
func Init(cfg Config) error {
	mu.Lock()
	defer mu.Unlock()

	resetLocked()

	if !cfg.Enabled {
		return nil
	}

	filePath := strings.TrimSpace(cfg.FilePath)
	if filePath == "" {
		filePath = defaultFilePath
	}
	basePath = filePath
	minLevel = parseLevel(cfg.Level)

	if err := openDailyFileLocked(nowFunc()); err != nil {
		resetLocked()
		return err
	}
	enabled = true
	return nil
}

// Enabled 返回文件日志是否已启用。
func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}

// CurrentFilePath 返回当前正在写入的日志文件路径；未启用时返回空字符串。
func CurrentFilePath() string {
	mu.Lock()
	defer mu.Unlock()
	if !enabled || logFile == nil {
		return ""
	}
	return logFile.Name()
}

// DailyFilePath 根据基路径与日期生成按天日志文件路径。
func DailyFilePath(base string, day time.Time) string {
	base = strings.TrimSpace(base)
	if base == "" {
		base = defaultFilePath
	}
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	if ext == "" {
		ext = ".log"
	}
	return fmt.Sprintf("%s-%s%s", name, day.Format(dailyDateLayout), ext)
}

// With 返回携带默认 attrs 的 Logger，便于固定 component 等字段。
func With(attrs ...any) *Logger {
	copied := make([]any, len(attrs))
	copy(copied, attrs)
	return &Logger{attrs: copied}
}

// Info 写入 INFO 级别 JSON 日志；未启用时为 no-op。
func Info(msg string, attrs ...any) {
	write(slog.LevelInfo, msg, attrs...)
}

// Warn 写入 WARN 级别 JSON 日志；未启用时为 no-op。
func Warn(msg string, attrs ...any) {
	write(slog.LevelWarn, msg, attrs...)
}

// Debug 写入 DEBUG 级别 JSON 日志；未启用时为 no-op。
func Debug(msg string, attrs ...any) {
	write(slog.LevelDebug, msg, attrs...)
}

// Error 写入 ERROR 级别 JSON 日志；未启用时为 no-op。
func Error(msg string, attrs ...any) {
	write(slog.LevelError, msg, attrs...)
}

// Info 写入 INFO 级别 JSON 日志，并附带 Logger 默认 attrs。
func (l *Logger) Info(msg string, attrs ...any) {
	write(slog.LevelInfo, msg, mergeAttrs(l, attrs)...)
}

// Warn 写入 WARN 级别 JSON 日志，并附带 Logger 默认 attrs。
func (l *Logger) Warn(msg string, attrs ...any) {
	write(slog.LevelWarn, msg, mergeAttrs(l, attrs)...)
}

// Debug 写入 DEBUG 级别 JSON 日志，并附带 Logger 默认 attrs。
func (l *Logger) Debug(msg string, attrs ...any) {
	write(slog.LevelDebug, msg, mergeAttrs(l, attrs)...)
}

// Error 写入 ERROR 级别 JSON 日志，并附带 Logger 默认 attrs。
func (l *Logger) Error(msg string, attrs ...any) {
	write(slog.LevelError, msg, mergeAttrs(l, attrs)...)
}

// Close 关闭日志文件句柄。
func Close() error {
	mu.Lock()
	defer mu.Unlock()
	return resetLocked()
}

// SetNowFunc 仅用于测试，覆盖当前时间来源。
func SetNowFunc(fn func() time.Time) {
	mu.Lock()
	defer mu.Unlock()
	if fn == nil {
		nowFunc = time.Now
		return
	}
	nowFunc = fn
}

func mergeAttrs(l *Logger, attrs []any) []any {
	if l == nil || len(l.attrs) == 0 {
		return attrs
	}
	if len(attrs) == 0 {
		return l.attrs
	}
	merged := make([]any, 0, len(l.attrs)+len(attrs))
	merged = append(merged, l.attrs...)
	merged = append(merged, attrs...)
	return merged
}

func write(level slog.Level, msg string, attrs ...any) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled {
		return
	}
	if err := ensureDailyFileLocked(nowFunc()); err != nil {
		return
	}
	if logger == nil {
		return
	}
	if level < minLevel {
		return
	}
	switch level {
	case slog.LevelWarn:
		logger.Warn(msg, attrs...)
	case slog.LevelError:
		logger.Error(msg, attrs...)
	case slog.LevelDebug:
		logger.Debug(msg, attrs...)
	default:
		logger.Info(msg, attrs...)
	}
}

func ensureDailyFileLocked(now time.Time) error {
	day := now.Format(dailyDateLayout)
	if logFile != nil && logger != nil && day == currentDay {
		return nil
	}
	return openDailyFileLocked(now)
}

func openDailyFileLocked(now time.Time) error {
	day := now.Format(dailyDateLayout)
	path := DailyFilePath(basePath, now)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create logging directory: %w", err)
	}

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open logging file %s: %w", path, err)
	}

	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}

	handler := slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: minLevel,
	})
	logger = slog.New(handler)
	logFile = file
	currentDay = day
	return nil
}

func resetLocked() error {
	enabled = false
	logger = nil
	minLevel = slog.LevelInfo
	basePath = ""
	currentDay = ""

	if logFile == nil {
		return nil
	}

	var err error
	if closeErr := logFile.Close(); closeErr != nil {
		err = closeErr
	}
	logFile = nil
	return err
}

func parseLevel(raw string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
