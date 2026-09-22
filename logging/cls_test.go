package logging

import (
	"log/slog"
	"testing"

	cls "github.com/tencentcloud/tencentcloud-cls-sdk-go"
)

type fakeCLSSender struct {
	topic string
	logs  []*cls.Log
	err   error
}

func (f *fakeCLSSender) SendLog(topicID string, log *cls.Log, _ cls.CallBack) error {
	f.topic = topicID
	f.logs = append(f.logs, log)
	return f.err
}

func TestCLSHandlerMapsFields(t *testing.T) {
	fake := &fakeCLSSender{}
	handler := newCLSHandler(fake, "topic-1", slog.LevelInfo)
	slog.New(handler).Info("hello", "component", "http", "user_id", "u1")

	if fake.topic != "topic-1" {
		t.Fatalf("topic = %q, want topic-1", fake.topic)
	}
	if len(fake.logs) != 1 {
		t.Fatalf("logs = %d, want 1", len(fake.logs))
	}
	got := clsContents(fake.logs[0])
	if got[FieldMsg] != "hello" {
		t.Fatalf("msg = %q, want hello", got[FieldMsg])
	}
	if got[FieldLevel] != "INFO" {
		t.Fatalf("level = %q, want INFO", got[FieldLevel])
	}
	if got[FieldTime] == "" {
		t.Fatal("missing time")
	}
	if got[FieldComponent] != "http" {
		t.Fatalf("component = %q, want http", got[FieldComponent])
	}
	if got[FieldUserID] != "u1" {
		t.Fatalf("user_id = %q, want u1", got[FieldUserID])
	}
}

func TestCLSHandlerDropsBelowLevel(t *testing.T) {
	fake := &fakeCLSSender{}
	slog.New(newCLSHandler(fake, "topic-1", slog.LevelInfo)).Debug("hidden")
	if len(fake.logs) != 0 {
		t.Fatalf("logs = %d, want 0", len(fake.logs))
	}
}

func TestInitCLSMissingTopic(t *testing.T) {
	t.Cleanup(func() { _ = Close() })

	err := Init(Config{
		Enabled: true,
		Driver:  DriverCLS,
		CLS: CLSConfig{
			Endpoint:        "ap-chengdu.cls.tencentyun.com",
			AccessKeyID:     "id",
			AccessKeySecret: "secret",
		},
	})
	if err == nil {
		t.Fatal("Init() error = nil, want missing topic")
	}
	if Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
}

func TestInitCLSMissingSecret(t *testing.T) {
	t.Cleanup(func() { _ = Close() })

	err := Init(Config{
		Enabled: true,
		Driver:  DriverCLS,
		CLS: CLSConfig{
			Endpoint:    "ap-chengdu.cls.tencentyun.com",
			TopicID:     "topic-1",
			AccessKeyID: "id",
		},
	})
	if err == nil {
		t.Fatal("Init() error = nil, want missing secret")
	}
}

func TestInitUnknownDriver(t *testing.T) {
	t.Cleanup(func() { _ = Close() })

	err := Init(Config{Enabled: true, Driver: "es"})
	if err == nil {
		t.Fatal("Init() error = nil, want invalid driver")
	}
}

func TestInitDisabledIgnoresCLS(t *testing.T) {
	t.Cleanup(func() { _ = Close() })

	if err := Init(Config{Enabled: false, Driver: DriverCLS}); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if Enabled() {
		t.Fatal("Enabled() = true, want false")
	}
}

func clsContents(log *cls.Log) map[string]string {
	out := make(map[string]string, len(log.GetContents()))
	for _, item := range log.GetContents() {
		out[item.GetKey()] = item.GetValue()
	}
	return out
}
