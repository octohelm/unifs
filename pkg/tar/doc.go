package tar

func WithImport(base string) ExportOption {
	return func(t *tarExporter) {
		t.base = base
	}
}
