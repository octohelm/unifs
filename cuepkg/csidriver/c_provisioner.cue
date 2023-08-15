package csidriver

import (
	kubepkgspec "github.com/octohelm/kubepkg/cuepkg/kubepkg"
)

#Provisioner: {
	#values: kubelet: root: string | *"/var/lib/k0s/kubelet"

	kubepkgspec.#KubePkg & {
		metadata: name: "csi-provisioner-unifs"

		spec: {
			version: _ | *"1.0.0"

			deploy: {
				kind: "StatefulSet"
				spec: serviceName: "csi-provisioner"

				spec: template: spec: tolerations: [
					{
						key:      "node-role.kubernetes.io/master"
						operator: "Exists"
					},
				]
			}

			containers: {
				"csi-provisioner": {
					image: {
						name: _ | *"registry.k8s.io/sig-storage/csi-provisioner"
						tag:  _ | *"v3.5.0"
					}

					args: [
						"--v=4",
						"--csi-address=$(ADDRESS)",
					]

					env: ADDRESS: "\(#values.kubelet.root)/plugins/\(#DriverName)/csi.sock"
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
						CSI_ENDPOINT: "unix://\(#values.kubelet.root)/plugins/\(#DriverName)/csi.sock"
						NODE_ID:      "@field/spec.nodeName"
					}
				}
			}

			volumes: "socket-dir": {
				type:      "EmptyDir"
				mountPath: "\(#values.kubelet.root)/plugins/\(#DriverName)"
			}

			serviceAccount: #ProvisionerServiceAccount
		}
	}
}

#ProvisionerServiceAccount: kubepkgspec.#ServiceAccount & {
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
				"persistentvolumes",
			]
			verbs: [
				"get",
				"list",
				"watch",
				"create",
				"delete",
			]
		},
		{
			apiGroups: [
				"",
			]
			resources: [
				"persistentvolumeclaims",
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
				"storageclasses",
			]
			verbs: [
				"get",
				"list",
				"watch",
			]
		},
		{
			apiGroups: [
				"",
			]
			resources: [
				"events",
			]
			verbs: [
				"list",
				"watch",
				"create",
				"update",
				"patch",
			]
		},
	]
}
