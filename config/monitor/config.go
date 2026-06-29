// Package monitor configures the groundcover_monitor_v2 resource as the Monitor kind. v2
// is fully typed (title, severity, query/threshold/reducer blocks, etc.), unlike the legacy
// groundcover_monitor which took an opaque monitor_yaml string — so the schema, validation,
// and marketplace docs are native rather than passthrough. v1 is intentionally not wired,
// so the provider no longer generates or manages it.
package monitor

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_monitor_v2 resource as the Monitor kind.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_monitor_v2", func(r *config.Resource) {
		r.ShortGroup = "monitoring"
		r.Kind = "Monitor"
		// display, query, evaluation_interval, and notification_settings are framework
		// SingleNestedAttributes (objects). The schema lift turns each into an SDKv2 singleton
		// list so upjet can build types; collapse them back to embedded objects so the CRD
		// fields and the runtime params are object-shaped, matching the provider. (threshold
		// and reducer are genuine lists and stay arrays.)
		if r.SchemaElementOptions == nil {
			r.SchemaElementOptions = config.SchemaElementOptions{}
		}
		for _, f := range []string{"display", "query", "evaluation_interval", "notification_settings"} {
			r.SchemaElementOptions.SetEmbeddedObject(f)
		}
	})
}
