//go:generate go tool gen .

// Package local 将本地目录暴露为 UniFS 的 FileSystem。
//
// 当前实现基于 golang.org/x/net/webdav.Dir，适合在测试、示例和本地文件场景中复用。
package local
