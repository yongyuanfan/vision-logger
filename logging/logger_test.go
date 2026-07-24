package logging_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	logging "github.com/yongyuanfan/vision-logger/logging"
)

func TestInitDisabled(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	dir := t.TempDir()
	logPath := filepath.Join(dir, "app.log")

	if err := logging.Init(logging.Config{Enabled: false, FilePath: logPath}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if logging.Enabled() {
		t.Fatal("Enabled() = true, want false")
	}

	logging.Info("should not write")
	dailyPath := logging.DailyFilePath(logPath, time.Now())
	if _, err := os.Stat(dailyPath); !os.IsNotExist(err) {
		t.Fatalf("log file should not exist when disabled, stat err = %v", err)
	}
}

func TestInitEnabledWritesJSON(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	dir := t.TempDir()
	basePath := filepath.Join(dir, "nested", "app.log")

	if err := logging.Init(logging.Config{
		Enabled:  true,
		Level:    "info",
		FilePath: basePath,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if !logging.Enabled() {
		t.Fatal("Enabled() = false, want true")
	}

	logging.Info("starting server", logging.FieldComponent, "runtime", logging.FieldService, "http")
	logging.Warn("tool call failed", logging.FieldComponent, "mcp-tool", "tool", "doc_generation")

	logPath := logging.CurrentFilePath()
	if logPath == "" {
		t.Fatal("CurrentFilePath() empty")
	}
	wantPath := logging.DailyFilePath(basePath, time.Now())
	if logPath != wantPath {
		t.Fatalf("CurrentFilePath() = %q, want %q", logPath, wantPath)
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2", len(lines))
	}

	var infoEntry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &infoEntry); err != nil {
		t.Fatalf("unmarshal info line: %v", err)
	}
	if infoEntry[logging.FieldLevel] != "INFO" {
		t.Fatalf("info level = %v, want INFO", infoEntry[logging.FieldLevel])
	}
	if infoEntry[logging.FieldMsg] != "starting server" {
		t.Fatalf("info msg = %v, want starting server", infoEntry[logging.FieldMsg])
	}
	if _, ok := infoEntry[logging.FieldTime]; !ok {
		t.Fatal("info missing time field")
	}
	if infoEntry[logging.FieldComponent] != "runtime" {
		t.Fatalf("info component = %v, want runtime", infoEntry[logging.FieldComponent])
	}
	if infoEntry[logging.FieldService] != "http" {
		t.Fatalf("info service = %v, want http", infoEntry[logging.FieldService])
	}

	var warnEntry map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &warnEntry); err != nil {
		t.Fatalf("unmarshal warn line: %v", err)
	}
	if warnEntry[logging.FieldLevel] != "WARN" {
		t.Fatalf("warn level = %v, want WARN", warnEntry[logging.FieldLevel])
	}
	if warnEntry["tool"] != "doc_generation" {
		t.Fatalf("warn tool = %v, want doc_generation", warnEntry["tool"])
	}
}

func TestInitInvalidLevelFallsBackToInfo(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	dir := t.TempDir()
	basePath := filepath.Join(dir, "app.log")

	if err := logging.Init(logging.Config{
		Enabled:  true,
		Level:    "unknown",
		FilePath: basePath,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	logging.Info("info message")
	logging.Warn("warn message")

	data, err := os.ReadFile(logging.CurrentFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "info message") || !strings.Contains(content, "warn message") {
		t.Fatalf("expected both info and warn messages in log, got: %s", content)
	}
}

func TestInitDefaultFilePathWhenEmpty(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir() error = %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	if err := logging.Init(logging.Config{Enabled: true}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	logging.Info("default path message")

	logPath := logging.DailyFilePath(filepath.Join(dir, "runtime", "logs", "app.log"), time.Now())
	if _, err := os.Stat(logPath); err != nil {
		t.Fatalf("default log file stat error = %v", err)
	}
}

func TestInitLevelFiltersDebug(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	dir := t.TempDir()
	basePath := filepath.Join(dir, "app.log")

	if err := logging.Init(logging.Config{
		Enabled:  true,
		Level:    "warn",
		FilePath: basePath,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	logging.Info("filtered info")
	logging.Warn("kept warn")

	data, err := os.ReadFile(logging.CurrentFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	if strings.Contains(content, "filtered info") {
		t.Fatalf("info message should be filtered at warn level, got: %s", content)
	}
	if !strings.Contains(content, "kept warn") {
		t.Fatalf("warn message should be kept, got: %s", content)
	}
}

func TestDebugAndErrorLevels(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	dir := t.TempDir()
	basePath := filepath.Join(dir, "app.log")

	if err := logging.Init(logging.Config{
		Enabled:  true,
		Level:    "debug",
		FilePath: basePath,
	}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	logging.Debug("debug message", logging.FieldComponent, "test")
	logging.Error("error message", logging.FieldComponent, "test")

	data, err := os.ReadFile(logging.CurrentFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "debug message") || !strings.Contains(content, "error message") {
		t.Fatalf("expected debug and error messages in log, got: %s", content)
	}
}

func TestReInitClosesPreviousFile(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	dir := t.TempDir()
	firstBase := filepath.Join(dir, "first.log")
	secondBase := filepath.Join(dir, "second.log")

	if err := logging.Init(logging.Config{Enabled: true, FilePath: firstBase}); err != nil {
		t.Fatalf("first Init() error = %v", err)
	}
	logging.Info("first")
	firstPath := logging.CurrentFilePath()

	if err := logging.Init(logging.Config{Enabled: true, FilePath: secondBase}); err != nil {
		t.Fatalf("second Init() error = %v", err)
	}
	logging.Info("second")
	secondPath := logging.CurrentFilePath()

	firstData, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("ReadFile(first) error = %v", err)
	}
	if !strings.Contains(string(firstData), "first") || strings.Contains(string(firstData), "second") {
		t.Fatalf("first log file content unexpected: %s", firstData)
	}

	secondData, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("ReadFile(second) error = %v", err)
	}
	if !strings.Contains(string(secondData), "second") {
		t.Fatalf("second log file content unexpected: %s", secondData)
	}
}

func TestDailyFilePath(t *testing.T) {
	day := time.Date(2026, 7, 9, 15, 4, 5, 0, time.Local)
	got := logging.DailyFilePath("runtime/logs/vision-ai.log", day)
	want := "runtime/logs/vision-ai-2026-07-09.log"
	if got != want {
		t.Fatalf("DailyFilePath() = %q, want %q", got, want)
	}

	got = logging.DailyFilePath("runtime/logs/app", day)
	want = "runtime/logs/app-2026-07-09.log"
	if got != want {
		t.Fatalf("DailyFilePath(no ext) = %q, want %q", got, want)
	}
}

func TestRotateToNextDay(t *testing.T) {
	t.Cleanup(func() {
		logging.SetNowFunc(time.Now)
		_ = logging.Close()
	})

	dir := t.TempDir()
	basePath := filepath.Join(dir, "app.log")
	day1 := time.Date(2026, 7, 9, 23, 59, 0, 0, time.Local)
	day2 := time.Date(2026, 7, 10, 0, 1, 0, 0, time.Local)

	logging.SetNowFunc(func() time.Time { return day1 })
	if err := logging.Init(logging.Config{Enabled: true, FilePath: basePath}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	logging.Info("day1 message")
	path1 := logging.CurrentFilePath()
	want1 := logging.DailyFilePath(basePath, day1)
	if path1 != want1 {
		t.Fatalf("day1 path = %q, want %q", path1, want1)
	}

	logging.SetNowFunc(func() time.Time { return day2 })
	logging.Info("day2 message")
	path2 := logging.CurrentFilePath()
	want2 := logging.DailyFilePath(basePath, day2)
	if path2 != want2 {
		t.Fatalf("day2 path = %q, want %q", path2, want2)
	}
	if path1 == path2 {
		t.Fatal("expected different log files across days")
	}

	day1Data, err := os.ReadFile(path1)
	if err != nil {
		t.Fatalf("ReadFile(day1) error = %v", err)
	}
	if !strings.Contains(string(day1Data), "day1 message") || strings.Contains(string(day1Data), "day2 message") {
		t.Fatalf("day1 content unexpected: %s", day1Data)
	}

	day2Data, err := os.ReadFile(path2)
	if err != nil {
		t.Fatalf("ReadFile(day2) error = %v", err)
	}
	if !strings.Contains(string(day2Data), "day2 message") {
		t.Fatalf("day2 content unexpected: %s", day2Data)
	}
}

func TestWithAndFromContext(t *testing.T) {
	t.Cleanup(func() { _ = logging.Close() })

	dir := t.TempDir()
	basePath := filepath.Join(dir, "app.log")
	if err := logging.Init(logging.Config{Enabled: true, Level: "info", FilePath: basePath}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	logging.With(logging.FieldComponent, "runtime").Info("with attrs", "k", "v")

	ctx := logging.ContextWithTraceID(context.Background(), "tid-1")
	ctx = logging.ContextWithUserID(ctx, "uid-1")
	logging.FromContext(ctx).Info("from context")

	data, err := os.ReadFile(logging.CurrentFilePath())
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d log lines, want 2", len(lines))
	}

	var withEntry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &withEntry); err != nil {
		t.Fatalf("unmarshal with line: %v", err)
	}
	if withEntry[logging.FieldComponent] != "runtime" || withEntry["k"] != "v" {
		t.Fatalf("with attrs unexpected: %v", withEntry)
	}

	var ctxEntry map[string]any
	if err := json.Unmarshal([]byte(lines[1]), &ctxEntry); err != nil {
		t.Fatalf("unmarshal context line: %v", err)
	}
	if ctxEntry[logging.FieldTraceID] != "tid-1" {
		t.Fatalf("trace_id = %v, want tid-1", ctxEntry[logging.FieldTraceID])
	}
	if ctxEntry[logging.FieldUserID] != "uid-1" {
		t.Fatalf("user_id = %v, want uid-1", ctxEntry[logging.FieldUserID])
	}
}
