package mounter

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mitchellh/go-ps"
	"k8s.io/utils/mount"

	"github.com/octohelm/unifs/pkg/filesystem/api"
	"github.com/octohelm/unifs/pkg/strfmt"
)

type Mounter interface {
	Mount(mountPoint string) error
}

func NewMounter(ctx context.Context, backendStr string) (Mounter, error) {
	backend, err := strfmt.ParseEndpoint(backendStr)
	if err != nil {
		return nil, fmt.Errorf("parse backend %q: %w", backendStr, err)
	}

	b := api.FileSystemBackend{}
	b.Backend = *backend

	// 仅用于提前校验后端参数。
	if err := b.Init(ctx); err != nil {
		return nil, fmt.Errorf("init backend %q: %w", backend.SecurityString(), err)
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
		return fmt.Errorf("mkdir mount point %q: %w", mountPoint, err)
	}

	p, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	args := []string{
		"mount",
		"--backend", m.Backend.String(),
		mountPoint,
	}

	cmd := exec.Command(p, args...)
	cmd.Env = os.Environ()

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("FuseMount: %s: %w", append([]string{p}, args...), err)
	}

	if err := waitForMount(mountPoint, 10*time.Second); err != nil {
		return fmt.Errorf("wait for mount %q: %w", mountPoint, err)
	}
	return nil
}

func FuseUnmount(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat mount path %q: %w", path, err)
	}

	m := mount.New("")

	notMount, err := m.IsLikelyNotMountPoint(path)
	if err != nil {
		return fmt.Errorf("check mount point %q: %w", path, err)
	}

	if notMount {
		return nil
	}

	if err := m.Unmount(path); err != nil {
		return fmt.Errorf("unmount %q: %w", path, err)
	}
	return nil
}

func waitForMount(path string, timeout time.Duration) error {
	var elapsed time.Duration
	interval := 10 * time.Millisecond
	for {
		notMount, err := mount.New("").IsLikelyNotMountPoint(path)
		if err != nil {
			return fmt.Errorf("check mount point %q: %w", path, err)
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
		return nil, fmt.Errorf("list processes: %w", err)
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
		return "", fmt.Errorf("read cmdline %q: %w", cmdLineFile, err)
	}
	return string(cmdLine), nil
}
