//go:generate go tool gen .

// Package logr 为 FileSystem 操作增加日志包装。
//
// Wrap 返回的文件系统会记录 mkdir、open、remove、rename 和 stat 等操作结果。
package logr
