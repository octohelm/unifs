//go:generate go tool gen .

// Package ftp 将 UniFS 的 FileSystem 暴露为 FTP 服务。
//
// 该包负责 FTP serverlib 适配、连接生命周期和优雅停止逻辑。
package ftp
