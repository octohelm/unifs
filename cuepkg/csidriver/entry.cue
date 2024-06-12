package csidriver

import (
	kubepkgspec "github.com/octohelm/kubepkgspec/cuepkg/kubepkg"
)

#Provider: {
	#values: {
		version: string | *"v1.0.0"

		provisioner: #Provisioner & {
			spec: "version": version
		}
		driver: #Driver & {
			spec: "version": version
		}
		attacher: #Attacher & {
			spec: "version": version
		}
	}

	kubepkgspec.#KubePkgList & {
		items: [
			#values.provisioner,
			#values.driver,
			#values.attacher,
		]
	}
}
