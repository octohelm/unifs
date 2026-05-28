package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/rhnvrm/simples3"

	"github.com/octohelm/unifs/pkg/filesystem"
	"github.com/octohelm/unifs/pkg/filesystem/fsutil"
)

type fs struct {
	s3Client           *simples3.S3
	s3ClientForPresign *simples3.S3
	httpClient         *http.Client
	presignAs          *url.URL

	bucket string
	prefix string
	region string
}

func (fsys *fs) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	if _, err := fsys.Stat(ctx, name); err == nil {
		return &os.PathError{
			Op:   "mkdir",
			Path: name,
			Err:  os.ErrExist,
		}
	}
	f, err := fsys.OpenFile(ctx, fmt.Sprintf("%s/", path.Clean(name)), os.O_CREATE, perm)
	if err != nil {
		return fmt.Errorf("open s3 dir %q: %w", name, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close s3 dir %q: %w", name, err)
	}
	return nil
}

func (fsys *fs) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (filesystem.File, error) {
	// S3 不支持追加写。
	if flag&os.O_APPEND != 0 {
		return nil, fmt.Errorf("file append is not allow: %w", os.ErrPermission)
	}

	if flag&os.O_CREATE != 0 {
		flag |= os.O_WRONLY
	}

	if strings.HasSuffix(name, "/") {
		return openDir(ctx, fsys, name)
	}

	if flag&os.O_WRONLY != 0 {
		return openFileForWrite(ctx, fsys, name, flag)
	}

	f, err := openFileForRead(ctx, fsys, name, flag)
	if err != nil {
		return nil, fmt.Errorf("open s3 object %q for read: %w", name, err)
	}
	return f, nil
}

func (fsys *fs) Rename(ctx context.Context, oldName, newName string) error {
	if newName == oldName {
		return nil
	}

	info, err := fsys.Stat(ctx, oldName)
	if err != nil {
		return fmt.Errorf("stat rename source %q: %w", oldName, err)
	}

	// /x 不能移动到 /x/a/b/x 这类子路径。
	if oldName == "/" || strings.HasPrefix(newName, oldName+"/") {
		return &os.LinkError{
			Op:  "rename",
			Old: oldName,
			New: newName,
			Err: os.ErrPermission,
		}
	}

	if info.IsDir() {
		f := &file{
			ctx:  ctx,
			fs:   fsys,
			name: oldName,
		}

		fileInfos, err := f.Readdir(0)
		if err != nil {
			return fmt.Errorf("readdir rename source %q: %w", oldName, err)
		}

		if err := fsys.Mkdir(ctx, newName, os.ModePerm); err != nil {
			return fmt.Errorf("mkdir rename destination %q: %w", newName, err)
		}

		for _, fi := range fileInfos {
			fullPath := path.Join(f.Name(), fi.Name())
			destFullPath := path.Join(newName, fi.Name())
			if err := fsys.Rename(ctx, fullPath, destFullPath); err != nil {
				return fmt.Errorf("rename child %q to %q: %w", fullPath, destFullPath, err)
			}
		}

		if err := fsys.forceRemove(ctx, oldName, true); err != nil {
			return fmt.Errorf("remove renamed source dir %q: %w", oldName, err)
		}
		return nil
	}

	_, err = fsys.s3Client.CopyObject(simples3.CopyObjectInput{
		SourceBucket: fsys.bucket,
		SourceKey:    fsys.path(oldName),
		DestBucket:   fsys.bucket,
		DestKey:      fsys.path(newName),
	})
	if err != nil {
		return fmt.Errorf("copy s3 object %q to %q: %w", oldName, newName, err)
	}

	if err := fsys.forceRemove(ctx, oldName, false); err != nil {
		return fmt.Errorf("remove renamed source file %q: %w", oldName, err)
	}
	return nil
}

func (fsys *fs) RemoveAll(ctx context.Context, name string) error {
	if name == "/" {
		return fmt.Errorf("rm '/' not allow: %w", os.ErrPermission)
	}

	f := &file{
		ctx:  ctx,
		fs:   fsys,
		name: name,
	}

	fileInfos, err := f.Readdir(0)
	if err != nil {
		return fmt.Errorf("readdir remove %q: %w", name, err)
	}

	for _, fi := range fileInfos {
		fullPath := path.Join(f.Name(), fi.Name())

		if fi.IsDir() {
			if err := fsys.RemoveAll(ctx, fullPath); err != nil {
				return fmt.Errorf("remove child dir %q: %w", fullPath, err)
			}
		} else {
			if err := fsys.forceRemove(ctx, fullPath, false); err != nil {
				return fmt.Errorf("remove child file %q: %w", fullPath, err)
			}
		}
	}

	if err := fsys.forceRemove(ctx, path.Clean(f.Name())+"/", true); err != nil {
		return fmt.Errorf("remove dir marker %q: %w", name, err)
	}

	return nil
}

func (fsys *fs) forceRemove(ctx context.Context, name string, isDir bool) error {
	if isDir {
		if err := fsys.s3Client.FileDelete(simples3.DeleteInput{
			Bucket:    fsys.bucket,
			ObjectKey: fsys.path(path.Join(name, dirHolder)),
		}); err != nil {
			return fmt.Errorf("remove s3 dir holder %q: %w", path.Join(name, dirHolder), err)
		}
	}

	if err := fsys.s3Client.FileDelete(simples3.DeleteInput{
		Bucket:    fsys.bucket,
		ObjectKey: fsys.path(name),
	}); err != nil {
		return fmt.Errorf("remove s3 object %q: %w", name, err)
	}
	return nil
}

func (fsys *fs) path(name string) (s string) {
	if fsys.prefix == "" || fsys.prefix == "/" {
		return strings.TrimPrefix(name, "/")
	}
	return strings.TrimPrefix(path.Join(fsys.prefix, name), "/")
}

func (fsys *fs) Stat(ctx context.Context, name string) (os.FileInfo, error) {
	if name == "/" || name == "." {
		return fsutil.NewDirFileInfo("/"), nil
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	info, err := fsys.s3Client.FileDetails(simples3.DetailsInput{
		Bucket:    fsys.bucket,
		ObjectKey: fsys.path(name),
	})
	if err != nil {
		if isS3NotFound(err) {
			return fsys.statDirectory(ctx, name)
		}

		return nil, &os.PathError{
			Op:   "stat",
			Path: name,
			Err:  os.ErrNotExist,
		}
	}

	size, err := strconv.ParseInt(info.ContentLength, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parse s3 object size %q: %w", info.ContentLength, err)
	}

	return fsutil.NewFileInfo(
		path.Base(name),
		size,
		parseS3Time(info.LastModified),
	), nil
}

func (fsys *fs) statDirectory(ctx context.Context, name string) (os.FileInfo, error) {
	nameClean := path.Clean(name)

	objects, prefixes, err := fsys.listObjects(ctx, simples3.ListInput{
		Bucket:  fsys.bucket,
		Prefix:  fsys.path(nameClean),
		MaxKeys: 1,
	})
	if err == nil && len(objects)+len(prefixes) > 0 {
		return fsutil.NewDirFileInfo(path.Base(name)), nil
	}

	return nil, &os.PathError{
		Op:   "stat",
		Path: name,
		Err:  os.ErrNotExist,
	}
}

func (fsys *fs) presignClient() *simples3.S3 {
	if presignAs := fsys.presignAs; presignAs != nil {

		if presignAs.User != nil {
			if pwd, ok := presignAs.User.Password(); ok && pwd == "fake" {
				return fsys.s3Client
			}
		}
		return fsys.s3ClientForPresign
	}

	return fsys.s3Client
}

func (fsys *fs) ensureBucket(ctx context.Context) error {
	resp, err := fsys.doPresignedRequest(ctx, fsys.s3Client, http.MethodHead, "", nil, nil, 0)
	if err != nil {
		return err
	}
	defer closeResponse(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusNotFound:
		_, _ = fsys.s3Client.CreateBucket(simples3.CreateBucketInput{
			Bucket: fsys.bucket,
			Region: fsys.region,
		})
		return nil
	default:
		return fmt.Errorf("status code: %s", resp.Status)
	}
}

func (fsys *fs) listObjects(ctx context.Context, input simples3.ListInput) ([]simples3.Object, []string, error) {
	var objects []simples3.Object
	var prefixes []string

	for {
		if err := ctx.Err(); err != nil {
			return nil, nil, err
		}

		resp, err := fsys.s3Client.List(input)
		if err != nil {
			return nil, nil, err
		}

		objects = append(objects, resp.Objects...)
		prefixes = append(prefixes, resp.CommonPrefixes...)

		if input.MaxKeys > 0 && int64(len(objects)+len(prefixes)) >= input.MaxKeys {
			return objects, prefixes, nil
		}
		if !resp.IsTruncated || resp.NextContinuationToken == "" {
			return objects, prefixes, nil
		}

		input.ContinuationToken = resp.NextContinuationToken
	}
}

func (fsys *fs) presignedURL(ctx context.Context, client *simples3.S3, method string, key string, headers http.Header) (*url.URL, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	extraHeaders := map[string]string{}
	for name, values := range headers {
		if len(values) > 0 {
			extraHeaders[name] = values[0]
		}
	}

	rawURL := client.GeneratePresignedURL(simples3.PresignedInput{
		Bucket:        fsys.bucket,
		ObjectKey:     key,
		Method:        method,
		Timestamp:     time.Now(),
		ExpirySeconds: 5 * 60,
		ExtraHeaders:  extraHeaders,
	})
	if rawURL == "" {
		return nil, fmt.Errorf("generate presigned %s url for %q failed", method, key)
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse presigned %s url for %q: %w", method, key, err)
	}
	return u, nil
}

func (fsys *fs) doPresignedRequest(ctx context.Context, client *simples3.S3, method string, key string, headers http.Header, body io.Reader, contentLength int64) (*http.Response, error) {
	u, err := fsys.presignedURL(ctx, client, method, key, headers)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	if contentLength >= 0 {
		req.ContentLength = contentLength
	}
	req.Header = headers.Clone()

	return fsys.client().Do(req)
}

func (fsys *fs) client() *http.Client {
	if fsys.httpClient != nil {
		return fsys.httpClient
	}
	return http.DefaultClient
}

func isS3NotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "status code: 404") ||
		strings.Contains(s, "404 Not Found") ||
		strings.Contains(s, "NoSuchKey") ||
		strings.Contains(s, "NoSuchBucket")
}

func parseS3Time(v string) time.Time {
	t, err := http.ParseTime(v)
	if err == nil {
		return t
	}
	t, err = time.Parse(time.RFC3339, v)
	if err == nil {
		return t
	}
	return time.Time{}
}

func closeResponse(resp *http.Response) {
	if resp == nil || resp.Body == nil {
		return
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
}

func (fsys *fs) presignForWrite() (*url.URL, bool) {
	if fsys.presignAs != nil {
		if fsys.presignAs.User != nil && fsys.presignAs.User.Username() == "rw" {
			return fsys.presignAs, true
		}
	}

	return nil, false
}

func (fsys *fs) presignForRead() (*url.URL, bool) {
	if fsys.presignAs != nil {
		return fsys.presignAs, true
	}
	return nil, false
}
