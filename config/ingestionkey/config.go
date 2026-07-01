// Package ingestionkey configures the groundcover_ingestionkey resource.
package ingestionkey

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_ingestionkey resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_ingestionkey", func(r *config.Resource) {
		r.ShortGroup = "rbac"
		r.Kind = "IngestionKey"
	})
}
