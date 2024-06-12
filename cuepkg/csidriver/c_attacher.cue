package csidriver

import (
	kubepkgspec "github.com/octohelm/kubepkgspec/cuepkg/kubepkg"
)

#Attacher: {
	#values: kubelet: root: string | *"/var/lib/k0s/kubelet"

	kubepkgspec.#KubePkg & {
		metadata: name: "csi-attacher-unifs"

		spec: {
			version: _ | *"1.0.0"

			deploy: {
				kind: "StatefulSet"
				spec: serviceName: "csi-attacher-unifs"

				spec: template: spec: tolerations: [
					{
						key:      "node-role.kubernetes.io/master"
						operator: "Exists"
					},
				]
			}

			containers: "csi-attacher": {
				image: {
					name: _ | *"registry.k8s.io/sig-storage/csi-attacher"
					tag:  _ | *"v4.3.0"
				}

				args: [
					"--csi-address=$(ADDRESS)",
				]

				env: ADDRESS: "\(#values.kubelet.root)/plugins/\(#DriverName)/csi.sock"
			}

			volumes: "socket-dir": {
				type:      "HostPath"
				mountPath: "\(#values.kubelet.root)/plugins/\(#DriverName)"
				opt: {
					path: "\(#values.kubelet.root)/plugins/\(#DriverName)"
					type: "DirectoryOrCreate"
				}
			}

			serviceAccount: #AttacherServiceAccount
		}
	}
}

#AttacherServiceAccount: kubepkgspec.#ServiceAccount & {
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
				"events",
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
				"",
			]
			resources: [
				"nodes",
			]
			verbs: [
				"get",
				"list",
				"watch",
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
				"patch",
			]
		},
		{
			apiGroups: [
				"storage.k8s.io",
			]
			resources: [
				"volumeattachments/status",
			]
			verbs: [
				"patch",
			]
		},
	]
}
