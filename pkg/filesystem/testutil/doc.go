//go:generate go tool gen .

// Package testutil 提供 FileSystem 后端的一致性测试工具。
//
// 新增或调整后端时，应优先复用 TestSimpleFS、TestFullFS 和 TestStandardFS
// 来验证基本读写语义与标准库 io/fs 适配行为。
package testutil
