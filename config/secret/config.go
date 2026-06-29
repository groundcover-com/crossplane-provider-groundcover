// Package secret configures the groundcover_secret resource.
package secret

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_secret resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_secret", func(r *config.Resource) {
		r.ShortGroup = "secrets"
		r.Kind = "Secret"
	})
}
