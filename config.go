package visionlogger

// Config 控制文件日志的启用、级别与落盘基路径。
type Config struct {
	// Enabled 表示是否启用 JSON 文件日志。
	Enabled bool
	// Level 日志级别：debug | info | warn | error，默认 info。
	Level string
	// FilePath 日志基路径，实际按天写入，例如 runtime/logs/app-2026-07-09.log。
	FilePath string
}
