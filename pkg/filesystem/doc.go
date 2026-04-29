//go:generate go tool gen .

// Package filesystem 定义 unifs 的核心文件系统抽象与标准库适配层。
//
// 本包保持既有接口兼容：FileSystem 直接等同于 golang.org/x/net/webdav.FileSystem，
// File 等同于 webdav.File，FileInfo 等同于 os.FileInfo。后端只要实现该接口，
// 就可以继续复用本包提供的读写工具、子树视图与标准库 io/fs 适配。
//
// 与标准库 io/fs 的关系如下：
//   - AsReadDirFS 将 FileSystem 暴露为 fs.ReadDirFS，用于 fs.ReadFile、fs.Stat、
//     fs.ReadDir、fs.WalkDir 等只读标准库入口。
//   - Sub 返回仍满足 FileSystem 的子树视图；配合 AsReadDirFS 可把可写后端的一段
//     目录树交给标准库只读消费者。
//   - ReadDir 与 WalkDir 按名称排序，行为对齐标准库遍历时对稳定顺序的预期。
//
// 路径约定尽量贴近 io/fs：Sub 内部使用 fs.ValidPath 校验相对路径，同时兼容传入
// 前导斜杠的历史调用；后端实现仍需要清楚处理自身不完全等同于 POSIX 文件系统的语义，
// 例如对象存储的目录、rename 或 append 限制。
package filesystem
