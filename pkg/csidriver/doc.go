//go:generate go tool gen .

// Package csidriver 提供基于 UniFS 后端的 CSI Driver 实现。
//
// 该包负责 CSI identity、controller 和 node 服务逻辑；命令行装配位于 internal/cmd/unifs。
package csidriver
