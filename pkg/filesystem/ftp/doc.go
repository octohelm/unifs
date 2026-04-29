//go:generate go tool gen .

// Package ftp 将 FTP 服务端适配为 UniFS 的 FileSystem 后端。
//
// 它负责 FTP 连接池、路径操作和错误归一化，对外仍暴露统一文件系统接口。
package ftp
