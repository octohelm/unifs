//go:generate go tool gen .

// Package units 提供 UniFS 配置中复用的单位类型。
//
// 当前主要包含 BinarySize，用于在文本配置与 Kubernetes resource.Quantity
// 之间转换存储容量。
package units
