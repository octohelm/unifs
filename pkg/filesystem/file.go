package filesystem

// FileTruncator 表示文件支持截断到指定大小。
type FileTruncator interface {
	Truncate(size int64) error
}

// FileSyncer 表示文件支持同步已写入内容。
type FileSyncer interface {
	Sync() error
}
