// Package monitor configures the groundcover_monitor_v2_json resource as the Monitor kind.
// It is the codegen-friendly sibling of groundcover_monitor_v2: identical except that
// notification_settings.connected_app_params is a JSON string instead of a map(object).
// upjet/SDKv2 has no map-of-object block, so on the typed resource that field is dropped
// during the schema lift (losing per-connected-app channel routing); the JSON string
// survives intact, so the JSON variant is the one wired here. v2 is fully typed otherwise
// (title, severity, query/threshold/reducer blocks, etc.), unlike the legacy
// groundcover_monitor which took an opaque monitor_yaml string. v1 is intentionally not
// wired, so the provider no longer generates or manages it.
package monitor

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_monitor_v2_json resource as the Monitor kind.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_monitor_v2_json", func(r *config.Resource) {
		r.ShortGroup = "monitoring"
		r.Kind = "Monitor"
		// display, query, evaluation_interval, and notification_settings are framework
		// SingleNestedAttributes (objects). The schema lift turns each into an SDKv2 singleton
		// list so upjet can build types; collapse them back to embedded objects so the CRD
		// fields and the runtime params are object-shaped, matching the provider. (threshold
		// and reducer are genuine lists and stay arrays.)
		//
		// SetEmbeddedObject does not recurse, so every nested SingleNestedBlock must be listed
		// by its dotted path (index segments stripped) — the same pattern as
		// config/synthetictest. Missing one leaves that field an array in the CRD while the
		// framework wants an object, so reconcile fails to decode it (e.g. query.rollup, which
		// only metricsql monitors populate — the first path to actually hit this).
		if r.SchemaElementOptions == nil {
			r.SchemaElementOptions = config.SchemaElementOptions{}
		}
		for _, f := range []string{
			"display",
			"query",
			"query.rollup",
			"query.relative_timerange",
			"evaluation_interval",
			"notification_settings",
			"threshold.custom_resolve_threshold",
			"threshold.relative_timerange",
			"reducer.relative_timerange",
		} {
			r.SchemaElementOptions.SetEmbeddedObject(f)
		}
	})
}
