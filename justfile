set allow-duplicate-variables := true

import? '.just/local.just'
import '.just/default.just'
import '.just/mod/go.just'

piper := 'TTY=0 piper -p piper.cue' + if env("DEBUG", "0") == '1' { ' --log-level=debug' } else { '' }
unifs := "go tool unifs"

ship:
    {{ piper }} do ship push

fmt-cue:
    cue fmt -s ./cuepkg/...

test-fuse:
    TEST_FUSE=1 \
    	go test -v -failfast ./pkg/fuse/...

mount-fs:
    {{ unifs }} mount --delegate \
    	--backend=file:///tmp/data /tmp/mnt

mount-webdav:
    {{ unifs }} mount \
    	--backend=${UNIFS_WEBDAV_ENDPOINT} /tmp/mnt

mount-s3:
    {{ unifs }} mount \
    	--backend=${UNIFS_S3_ENDPOINT} /tmp/mnt

serve-webdav:
    {{ unifs }} webdav --backend=file:///tmp/data

serve-ftp:
    {{ unifs }} ftp --backend=file:///tmp/data

test-remote-s3:
    TEST_S3_ENDPOINT=${UNIFS_S3_ENDPOINT} \
    	go test -v -failfast ./pkg/filesystem/s3/...

test-remote-webdav:
    TEST_WEBDAV_ENDPOINT=${UNIFS_WEBDAV_ENDPOINT} \
    	go test -v -failfast ./pkg/filesystem/webdav/...

debug-apply:
    kubectl apply -f .tmp/manifests/unifs.yaml
