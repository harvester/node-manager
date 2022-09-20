package main

import (
	"os"

	controllergen "github.com/rancher/wrangler/pkg/controller-gen"
	"github.com/rancher/wrangler/pkg/controller-gen/args"

	ksmtunedv1 "github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func main() {
	os.Unsetenv("GOPATH")
	controllergen.Run(args.Options{
		OutputPackage: "github.com/harvester/node-manager/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			ksmtunedv1.GroupName: {
				Types: []interface{}{
					ksmtunedv1.Ksmtuned{},
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
		},
	})
}
