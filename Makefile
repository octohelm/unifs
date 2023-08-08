ifneq ( ,$(wildcard .secrets/local.mk))
	include .secrets/local.mk
endif

UNIFS = go run ./cmd/unifs

dep:
	go get -u ./pkg/...

test:
	go test -v ./pkg/...

test.fuse:
	TEST_FUSE=1 \
		go test -v -failfast ./pkg/fuse/...

mount.fs:
	UNIFS_ENDPOINT=file:///tmp/data \
		$(UNIFS) mount /tmp/mnt

mount.webdav:
	UNIFS_ENDPOINT=$(UNIFS_WEBDAV_ENDPOINT) \
		$(UNIFS) mount /tmp/mnt

mount.s3:
	UNIFS_ENDPOINT=$(UNIFS_S3_ENDPOINT) \
		$(UNIFS) mount /tmp/mnt

serve.webdav:
	UNIFS_ENDPOINT=file:///tmp/data \
		$(UNIFS) webdav


test.remote.s3:
	TEST_S3_ENDPOINT=$(UNIFS_S3_ENDPOINT) \
		go test -v -failfast ./pkg/filesystem/s3/...

test.remote.webdav:
	TEST_WEBDAV_ENDPOINT=$(UNIFS_WEBDAV_ENDPOINT) \
		go test -v -failfast ./pkg/filesystem/webdav/...
