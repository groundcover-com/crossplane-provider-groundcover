// Command generator runs the upjet code-generation pipeline for the groundcover
// Crossplane provider. It builds the provider configuration (which embeds the Terraform
// provider schema; see config/schema.go) and generates the CRD API types, controllers,
// and example manifests under the module root.
//
// It expects config/schema.json, produced by `make schema`
// (`terraform providers schema -json` against the groundcover provider).
//
// Invoke via `make generate`.
package main

import (
	"os"

	"github.com/crossplane/upjet/pkg/pipeline"

	"github.com/groundcover-com/crossplane-provider-groundcover/config"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	pipeline.Run(config.GetProvider(), root)
}
