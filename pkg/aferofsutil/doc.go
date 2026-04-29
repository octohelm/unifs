//go:generate go tool gen .

// Package aferofsutil 提供 FileSystem 到 afero.Fs 的适配。
//
// 该包用于把 UniFS 后端交给依赖 github.com/spf13/afero 的调用方。
package aferofsutil
