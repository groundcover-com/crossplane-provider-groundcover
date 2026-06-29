// Package tracespipeline configures the groundcover_tracespipeline resource.
package tracespipeline

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_tracespipeline resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_tracespipeline", func(r *config.Resource) {
		r.ShortGroup = "pipelines"
		r.Kind = "TracesPipeline"
	})
}
