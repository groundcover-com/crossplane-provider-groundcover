// Package metricsaggregation configures the groundcover_metricsaggregation resource.
package metricsaggregation

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_metricsaggregation resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_metricsaggregation", func(r *config.Resource) {
		r.ShortGroup = "pipelines"
		r.Kind = "MetricsAggregation"
	})
}
