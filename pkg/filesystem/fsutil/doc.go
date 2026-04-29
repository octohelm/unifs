//go:generate go tool gen .

// Package fsutil 提供构造 FileInfo 的小型辅助函数。
//
// 后端在没有本地 os.FileInfo 可复用时，可用本包生成目录或普通文件的信息对象。
package fsutil
