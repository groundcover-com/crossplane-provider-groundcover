// Package apikey configures the groundcover_apikey resource.
package apikey

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_apikey resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_apikey", func(r *config.Resource) {
		r.ShortGroup = "rbac"
		r.Kind = "APIKey"
	})
}
