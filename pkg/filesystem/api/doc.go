//go:generate go tool gen .

// Package api 提供文件系统后端的配置初始化入口。
//
// FileSystemBackend 根据 endpoint 选择具体后端，并把初始化后的 FileSystem 注入 context。
package api
