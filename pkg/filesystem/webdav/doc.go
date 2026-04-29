//go:generate go tool gen .

// Package webdav 将 WebDAV 服务适配为 UniFS 的 FileSystem 后端。
//
// 它基于 pkg/filesystem/webdav/client 发起 WebDAV 请求，并对外提供统一文件系统接口。
package webdav
