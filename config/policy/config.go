package policy

import "github.com/crossplane/upjet/pkg/config"

func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_policy", func(r *config.Resource) {
		r.ShortGroup = "rbac"
		r.Kind = "Policy"

		// data_scope and its nested groups are terraform-plugin-framework
		// SingleNestedAttributes (objects). liftNestedAttributesToBlocks turns them into
		// singleton-list blocks for upjet's SDKv2 converter; SetEmbeddedObject collapses
		// them back to embedded objects so the runtime value matches the framework
		// provider's object type. conditions/filters stay lists (ListNestedAttribute).
		if r.SchemaElementOptions == nil {
			r.SchemaElementOptions = config.SchemaElementOptions{}
		}
		for _, f := range []string{
			"data_scope",
			"data_scope.simple",
			"data_scope.advanced",
			"data_scope.advanced.events",
			"data_scope.advanced.logs",
			"data_scope.advanced.metrics",
			"data_scope.advanced.traces",
			"data_scope.advanced.workloads",
		} {
			r.SchemaElementOptions.SetEmbeddedObject(f)
		}
	})
}
