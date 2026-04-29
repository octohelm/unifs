//go:generate go tool gen .

// Package s3 将 S3 兼容对象存储适配为 UniFS 的 FileSystem 后端。
//
// 该包处理对象路径、目录占位对象、预签名读写和对象存储错误到文件系统错误的映射。
package s3
