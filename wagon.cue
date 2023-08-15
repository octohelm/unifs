package main

import (
	"strings"

	"wagon.octohelm.tech/core"
	"github.com/innoai-tech/runtime/cuepkg/debian"
	"github.com/innoai-tech/runtime/cuepkg/golang"

	"github.com/octohelm/unifs/cuepkg/csidriver"
	"github.com/octohelm/unifs/cuedevpkg/tool"
)

pkg: version: core.#Version & {
}

actions: go: golang.#Project & {
	source: {
		path: "."
		include: [
			"cmd/",
			"internal/",
			"pkg/",
			"go.mod",
			"go.sum",
		]
	}

	version: pkg.version.output

	goos: ["linux"]
	goarch: ["amd64", "arm64"]
	main: "./cmd/unifs"
	ldflags: [
		"-s -w",
		"-X \(go.module)/internal/version.version=\(go.version)",
	]

	build: pre: [
		"go mod download",
	]

	ship: {
		name: "\(strings.Replace(go.module, "github.com/", "ghcr.io/", -1))"
		tag:  pkg.version.output
		from: "docker.io/library/debian:bookworm-slim"

		steps: [
			debian.#InstallPackage & {
				packages: "fuse3": _
			},
		]
		config: cmd: ["csidriver"]
	}
}

actions: export: tool.#Export & {
	name:      "unifs"
	namespace: "storage-system--unifs"
	kubepkg:   csidriver.#Provider & {
		#values: version: pkg.version.output
	}
}

setting: {
	_env: core.#ClientEnv & {
		GH_USERNAME: string | *""
		GH_PASSWORD: core.#Secret
	}

	setup: core.#Setting & {
		registry: "ghcr.io": auth: {
			username: _env.GH_USERNAME
			secret:   _env.GH_PASSWORD
		}
	}
}
