// Package logspipeline configures the groundcover_logspipeline resource.
package logspipeline

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_logspipeline resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_logspipeline", func(r *config.Resource) {
		r.ShortGroup = "pipelines"
		r.Kind = "LogsPipeline"
	})
}
