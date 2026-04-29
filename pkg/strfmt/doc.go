//go:generate go tool gen .

// Package strfmt 提供 UniFS 配置中使用的字符串格式类型。
//
// Endpoint 用于解析后端 endpoint，并在日志或错误中隐藏敏感凭据。
package strfmt
