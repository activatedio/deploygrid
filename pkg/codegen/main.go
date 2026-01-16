package main

import (
	controllergen "github.com/rancher/wrangler/v3/pkg/controller-gen"
	"github.com/rancher/wrangler/v3/pkg/controller-gen/args"
)

func main() {

	controllergen.Run(args.Options{
		OutputPackage: "github.com/activatedio/deploygrid/pkg/generated",
		Boilerplate:   "hack/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"deploygrid.activated.io": {
				PackageName: "deploygrid.activated.io",
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					"./pkg/apis/deploygrid.activated.io/v1alpha1",
				},
				GenerateTypes:   true,
				GenerateClients: true,
			},
		},
	})

}
