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
	"path/filepath"
	"strings"

	"github.com/crossplane/upjet/pkg/pipeline"

	"github.com/groundcover-com/crossplane-provider-groundcover/config"
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	pipeline.Run(config.GetProvider(), root)
	injectQueryDataTypeEnum(root)
}

// injectQueryDataTypeEnum pins an Enum marker onto Monitor query.dataType so the CRD rejects
// unsupported values (notably `rum`, which the UI can't render). The Terraform provider enforces
// this in ValidateConfig, but upjet's plugin-framework client bypasses ValidateConfig, so the
// restriction has to live on the CRD. upjet offers no config hook to add a forProvider enum
// (SchemaElementOption only covers embedded-object/observation/init-provider), so we inject the
// marker into the freshly generated types before controller-gen reads them.
//
// ponytail: keyed on the exact generated field line. If a schema change renames it, the guard
// panics (fails the build) rather than silently dropping enforcement — fix the constant then.
func injectQueryDataTypeEnum(root string) {
	path := filepath.Join(root, "apis", "monitoring", "v1alpha1", "zz_monitor_types.go")
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	const field = "\tDataType *string `json:\"dataType,omitempty\" tf:\"data_type,omitempty\"`"
	const marker = "\t// +kubebuilder:validation:Enum=logs;traces;events;apm\n"
	src := string(b)
	if !strings.Contains(src, field) {
		panic("generator: Monitor query.DataType field not found; cannot inject dataType enum (did the schema change?)")
	}
	if err := os.WriteFile(path, []byte(strings.ReplaceAll(src, field, marker+field)), 0o644); err != nil {
		panic(err)
	}
}
