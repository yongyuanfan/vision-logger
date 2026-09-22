package logging

// 日志输出驱动。同一时间只启用一种。
const (
	// DriverFile 按天写入本地 NDJSON 文件。空值也按此处理。
	DriverFile = "file"
	// DriverCLS 异步上报腾讯云 CLS。
	DriverCLS = "cls"
)

// Config 控制日志启用、级别与输出驱动。
type Config struct {
	// Enabled 为 false 时不写任何日志。
	Enabled bool
	// Driver 取值 file 或 cls；空值按 file。
	Driver string
	// Level 日志级别：debug | info | warn | error，默认 info。
	Level string
	// FilePath 仅 driver=file 时使用。日志基路径，实际按天写入，例如 runtime/logs/app-2026-07-09.log。
	FilePath string
	// CLS 仅 driver=cls 时使用。
	CLS CLSConfig
}

// CLSConfig 腾讯云 CLS 异步上报参数。密钥由调用方从环境变量注入，本包不读取环境。
type CLSConfig struct {
	// Endpoint 地域接入域名，例如内网 ap-chengdu.cls.tencentyun.com。
	Endpoint string
	// TopicID 日志主题 ID。
	TopicID string
	// AccessKeyID 与 AccessKeySecret 为 CAM 永久密钥。
	AccessKeyID string
	// AccessKeySecret 与 AccessKeyID 成对出现。
	AccessKeySecret string
	// AccessToken 可选，仅 STS 临时凭证需要。
	AccessToken string
}
