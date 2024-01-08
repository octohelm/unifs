package filesystem

type FileTruncator interface {
	Truncate(size int64) error
}

type FileSyncer interface {
	Sync() error
}
