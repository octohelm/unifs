//go:generate go tool gen .

// Package mounter 提供 CSI NodePublishVolume 使用的挂载辅助能力。
//
// 它负责校验后端配置、启动 unifs mount 子进程，并在卸载时定位相关 FUSE 挂载。
package mounter
