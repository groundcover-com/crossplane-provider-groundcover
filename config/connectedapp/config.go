// Package connectedapp configures the groundcover_connected_app_json resource. Its data is
// a JSON string (the codegen-friendly sibling of the dynamic groundcover_connected_app),
// sensitive and redacted on read, so drift is detected via a server-computed data_hash.
// That suppression lives in the provider's Read and carries through upjet, so no
// controller-layer decorator is required.
package connectedapp

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_connected_app_json resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_connected_app_json", func(r *config.Resource) {
		r.ShortGroup = "integrations"
		r.Kind = "ConnectedAppJson"
	})
}
