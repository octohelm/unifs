//go:generate go tool gen .

// Package tar 提供 FileSystem 与 tar 归档之间的导入导出工具。
//
// From 创建导出器，把 FileSystem 中的文件写入 tar 流；To 创建导入器，
// 把 tar 流写回 FileSystem。目录创建和文件打开都复用 pkg/filesystem 的接口语义。
package tar

func WithImport(base string) ExportOption {
	return func(t *tarExporter) {
		t.base = base
	}
}
