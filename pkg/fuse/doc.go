//go:generate go tool gen .

// Package fuse 将 FileSystem 适配为 go-fuse 节点树。
//
// 该包只负责挂载层桥接，具体存储语义仍由传入的 FileSystem 后端决定。
package fuse
