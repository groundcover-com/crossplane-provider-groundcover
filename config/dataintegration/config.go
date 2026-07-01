// Package dataintegration configures the groundcover_dataintegration resource.
package dataintegration

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_dataintegration resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_dataintegration", func(r *config.Resource) {
		r.ShortGroup = "integrations"
		r.Kind = "DataIntegration"
	})
}
