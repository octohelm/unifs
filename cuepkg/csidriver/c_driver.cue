package csidriver

import (
	kubepkgspec "github.com/octohelm/kubepkg/cuepkg/kubepkg"
)

#Driver: {
	#values: {
		kubelet: root: string | *"/var/lib/k0s/kubelet"
		pods: root:    string | *"/data/k0s/kubelet/pods"
	}

	kubepkgspec.#KubePkg & {
		metadata: name: "csi-driver-unifs"

		spec: {
			version: _ | *"1.0.0"

			deploy: {
				kind: "DaemonSet"
				spec: template: spec: {
					hostNetwork: true
					dnsPolicy:   "ClusterFirstWithHostNet"
				}
			}

			containers: {
				"driver-registrar": {
					image: {
						// https://kubernetes-csi.github.io/docs/node-driver-registrar.html
						name: _ | *"registry.k8s.io/sig-storage/csi-node-driver-registrar"
						tag:  _ | *"v2.8.0"
					}

					args: [
						"--kubelet-registration-path=$(DRIVER_REG_SOCK_PATH)",
						"--csi-address=$(ADDRESS)",
					]

					env: {
						ADDRESS:              "/csi/csi.sock"
						DRIVER_REG_SOCK_PATH: "\(#values.kubelet.root)/plugins/\(#DriverName)/csi.sock"
						KUBE_NODE_NAME:       "@field/spec.nodeName"
					}

					securityContext: {
						privileged: true
						capabilities: add: ["SYS_ADMIN"]
						allowPrivilegeEscalation: true
					}
				}
				"csi-driver": {
					image: {
						name: _ | *"ghcr.io/octohelm/unifs"
						tag:  _ | *"\(spec.version)"
					}
					args: [
						"csidriver",
						"--endpoint=$(CSI_ENDPOINT)",
						"--nodeid=$(NODE_ID)",
					]
					env: {
						CSI_ENDPOINT: "unix:///csi/csi.sock"
						NODE_ID:      "@field/spec.nodeName"
					}
					securityContext: {
						privileged: true
						capabilities: add: ["SYS_ADMIN"]
						allowPrivilegeEscalation: true
					}
				}
			}

			volumes: {
				"registration-dir": {
					mountPath: "/registration/"
					type:      "HostPath"
					opt: path: "\(#values.kubelet.root)/plugins_registry/"
				}

				"plugin-dir": {
					mountPath: "/csi"
					type:      "HostPath"
					opt: path: "\(#values.kubelet.root)/plugins/\(#DriverName)"
				}

				"pods-dir": {
					mountPath:        "\(#values.pods.root)"
					mountPropagation: "Bidirectional"
					type:             "HostPath"
					opt: path: "\(#values.pods.root)"
				}

				"fuse-device": {
					type:      "HostPath"
					mountPath: "/dev/fuse"
					opt: path: "/dev/fuse"
				}
				data: {
					mountPath: "/data"
					type:      "HostPath"
					opt: path: "/data"
				}
			}

			serviceAccount: #DriverServiceAccount
		}
	}
}

#DriverServiceAccount: kubepkgspec.#ServiceAccount & {
	scope: "Cluster"
	rules: [
		{
			apiGroups: [
				"",
			]
			resources: [
				"secrets",
			]
			verbs: [
				"get",
				"list",
			]
		},
		{
			apiGroups: [
				"",
			]
			resources: [
				"nodes",
			]
			verbs: [
				"get",
				"list",
				"update",
			]
		},
		{
			apiGroups: [
				"",
			]
			resources: [
				"namespaces",
			]
			verbs: [
				"get",
				"list",
			]
		},
		{
			apiGroups: [
				"",
			]
			resources: [
				"persistentvolumes",
			]
			verbs: [
				"get",
				"list",
				"watch",
				"update",
			]
		},
		{
			apiGroups: [
				"storage.k8s.io",
			]
			resources: [
				"volumeattachments",
			]
			verbs: [
				"get",
				"list",
				"watch",
				"update",
			]
		},
	]
}
