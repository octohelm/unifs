package mounter

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/mitchellh/go-ps"
	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/strfmt"
	"k8s.io/utils/mount"
)

type Mounter interface {
	Mount(mountPoint string) error
}

func NewMounter(ctx context.Context, backendStr string) (Mounter, error) {
	backend, err := strfmt.ParseEndpoint(backendStr)
	if err != nil {
		return nil, err
	}

	b := api.FileSystemBackend{}
	b.Backend = *backend

	// just for param check
	if err := b.Init(ctx); err != nil {
		return nil, err
	}

	return &mounter{
		Backend: b.Backend,
	}, nil
}

type mounter struct {
	Backend strfmt.Endpoint
}

func (m *mounter) Mount(mountPoint string) error {
	if err := os.MkdirAll(mountPoint, os.ModeDir); err != nil {
		return err
	}

	p, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{
		"mount",
		"--backend", m.Backend.String(),
		mountPoint,
	}

	cmd := exec.Command(p, args...)
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "FuseMount: %s\n", append([]string{p}, args...))
	}

	return waitForMount(mountPoint, 10*time.Second)
}

func FuseUnmount(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	m := mount.New("")

	notMount, err := m.IsLikelyNotMountPoint(path)
	if err != nil {
		return err
	}

	if notMount {
		return nil
	}

	if err := m.Unmount(path); err != nil {
		return err
	}
	return nil
}

func waitForMount(path string, timeout time.Duration) error {
	var elapsed time.Duration
	var interval = 10 * time.Millisecond
	for {
		notMount, err := mount.New("").IsLikelyNotMountPoint(path)
		if err != nil {
			return err
		}
		if !notMount {
			return nil
		}
		time.Sleep(interval)
		elapsed = elapsed + interval
		if elapsed >= timeout {
			return errors.New("Timeout waiting for mount")
		}
	}
}

func FindFuseMountProcess(path string) (*os.Process, error) {
	processes, err := ps.Processes()
	if err != nil {
		return nil, err
	}
	for _, p := range processes {
		cmdLine, err := getCmdLine(p.Pid())
		if err != nil {
			continue
		}
		if strings.Contains(cmdLine, path) {
			return os.FindProcess(p.Pid())
		}
	}
	return nil, nil
}

func getCmdLine(pid int) (string, error) {
	cmdLineFile := fmt.Sprintf("/proc/%v/cmdline", pid)
	cmdLine, err := os.ReadFile(cmdLineFile)
	if err != nil {
		return "", err
	}
	return string(cmdLine), nil
}
