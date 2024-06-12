ifneq ( ,$(wildcard .secrets/local.mk))
	include .secrets/local.mk
endif

PIPER = TTY=0 piper -p piper.cue

DEBUG = 0
ifeq ($(DEBUG),1)
	PIPER := $(PIPER) --log-level=debug
endif

UNIFS = go run ./cmd/unifs

gen:
	go run ./internal/cmd/tool gen ./cmd/unifs

ship:
	$(PIPER) do ship push

fmt:
	cue fmt -s ./cuepkg/...
	cue fmt -s ./cuedevpkg/...
	goimports -w ./pkg
	goimports -w ./cmd

dep:
	go get -u ./pkg/...

test:
	go test -v -failfast ./pkg/...

install:
	go install ./cmd/unifs

test.fuse:
	TEST_FUSE=1 \
		go test -v -failfast ./pkg/fuse/...

mount.fs: install
	unifs mount --delegate \
		--backend=file:///tmp/data /tmp/mnt

mount.webdav:
	$(UNIFS) mount \
		--backend=$(UNIFS_WEBDAV_ENDPOINT) /tmp/mnt

mount.s3:
	$(UNIFS) mount \
		--backend=$(UNIFS_S3_ENDPOINT) /tmp/mnt

serve.webdav: install
	unifs webdav --backend=file:///tmp/data

serve.ftp: install
	unifs ftp --backend=file:///tmp/data

test.remote.s3:
	TEST_S3_ENDPOINT=$(UNIFS_S3_ENDPOINT) \
		go test -v -failfast ./pkg/filesystem/s3/...

test.remote.webdav:
	TEST_WEBDAV_ENDPOINT=$(UNIFS_WEBDAV_ENDPOINT) \
		go test -v -failfast ./pkg/filesystem/webdav/...

debug.apply:
	KUBECONFIG=${HOME}/.kube_config/config--infra-staging.yaml kubectl apply -f .tmp/manifests/unifs.yaml