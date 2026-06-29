// Package metricspipeline configures the groundcover_metricspipeline resource. Its "rules"
// field is a framework SingleNestedAttribute; the schema lift turns it into an SDKv2
// singleton list, so collapse it back to an embedded object (same pattern as
// notification_route.notification_settings) to match the provider's object-typed attribute.
package metricspipeline

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_metricspipeline resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_metricspipeline", func(r *config.Resource) {
		r.ShortGroup = "pipelines"
		r.Kind = "MetricsPipeline"
		if r.SchemaElementOptions == nil {
			r.SchemaElementOptions = config.SchemaElementOptions{}
		}
		r.SchemaElementOptions.SetEmbeddedObject("rules")
	})
}
