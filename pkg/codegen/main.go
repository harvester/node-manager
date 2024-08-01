package main

import (
	"os"

	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"

	nodev1beta1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func main() {
	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/harvester/node-manager/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"node.harvesterhci.io": {
				Types: []interface{}{
					nodev1beta1.Ksmtuned{},
					nodev1beta1.NodeConfig{},
					nodev1beta1.CloudInit{},
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
		},
	})
}
