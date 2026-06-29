// Package silence configures the groundcover_silence resource.
package silence

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_silence resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_silence", func(r *config.Resource) {
		r.ShortGroup = "monitoring"
		r.Kind = "Silence"
	})
}
