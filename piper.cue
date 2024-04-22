package main

import (
	"strings"

	"piper.octohelm.tech/wd"
	"piper.octohelm.tech/client"
	"piper.octohelm.tech/container"

	"github.com/octohelm/piper/cuepkg/debian"
	"github.com/octohelm/piper/cuepkg/golang"
	"github.com/octohelm/piper/cuepkg/containerutil"
)

hosts: {
	local: wd.#Local & {
	}
}

pkg: {
	_ver: client.#RevInfo & {
	}

	version: _ver.version
}

actions: go: golang.#Project & {
	cwd: hosts.local.dir

	version: pkg.version

	goos: ["linux"]
	goarch: ["amd64", "arm64"]
	main: "./cmd/unifs"
	ldflags: [
		"-s -w",
		"-X \(go.module)/internal/version.version=\(go.version)",
	]
	env: {
		GOEXPERIMENT: "rangefunc"
	}

}

actions: ship: containerutil.#Ship & {
	name: "\(strings.Replace(actions.go.module, "github.com/", "ghcr.io/", -1))"
	tag:  pkg.version
	from: "docker.io/library/debian:bookworm-slim"

	steps: [
		debian.#InstallPackage & {
			packages: "fuse3": _
		},
		container.#Set & {
			 config: cmd: ["csidriver"]
		},
	]
}

settings: {
	_env: client.#Env & {
		GH_USERNAME!: string
		GH_PASSWORD!: client.#Secret
	}

	registry: container.#Config & {
		auths: "ghcr.io": {
			username: _env.GH_USERNAME
			secret:   _env.GH_PASSWORD
		}
	}
}
