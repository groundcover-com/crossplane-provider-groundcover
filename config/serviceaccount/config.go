// Package serviceaccount configures the groundcover_serviceaccount resource.
package serviceaccount

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_serviceaccount resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_serviceaccount", func(r *config.Resource) {
		r.ShortGroup = "rbac"
		r.Kind = "ServiceAccount"
	})
}
