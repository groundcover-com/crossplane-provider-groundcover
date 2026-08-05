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
	"bytes"
	"os"
	"path/filepath"

	"github.com/crossplane/upjet/pkg/pipeline"

	"github.com/groundcover-com/crossplane-provider-groundcover/config"
)

var (
	metricRecorderCall = []byte("metrics.NewMetricRecorder(")
	metricsImport      = []byte("\t\"github.com/crossplane/upjet/pkg/metrics\"\n")
	handlerImport      = []byte("\t\"github.com/crossplane/upjet/pkg/controller/handler\"\n")
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	pipeline.Run(config.GetProvider(), root)
	if err := injectMetricsImport(root); err != nil {
		panic(err)
	}
}

// injectMetricsImport works around an upjet v1.11.0 template issue: generated controllers
// call metrics.NewMetricRecorder but omit the corresponding import.
func injectMetricsImport(root string) error {
	return filepath.WalkDir(filepath.Join(root, "internal", "controller"), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "zz_controller.go" {
			return nil
		}

		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Contains(b, metricRecorderCall) || bytes.Contains(b, metricsImport) {
			return nil
		}

		next := bytes.Replace(b, handlerImport, bytes.Join([][]byte{handlerImport, metricsImport}, nil), 1)
		return os.WriteFile(path, next, 0o644)
	})
}
