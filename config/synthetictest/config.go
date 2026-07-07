package synthetictest

import "github.com/crossplane/upjet/pkg/config"

func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_synthetic_test", func(r *config.Resource) {
		r.ShortGroup = "monitoring"
		r.Kind = "SyntheticTest"

		// http_check/ssl_check/tcp_check/dns_check/retry/monitor and their nested
		// single blocks are terraform-plugin-framework SingleNestedBlocks (objects).
		// upjet's SDKv2 view renders single-nesting as a singleton list; SetEmbeddedObject
		// collapses each back to an embedded object so the runtime value matches the
		// framework provider's object type. `assertion` is a ListNestedBlock and stays a
		// list, so it is intentionally omitted here.
		if r.SchemaElementOptions == nil {
			r.SchemaElementOptions = config.SchemaElementOptions{}
		}
		for _, f := range []string{
			"http_check",
			"http_check.body",
			"http_check.auth",
			"ssl_check",
			"tcp_check",
			"dns_check",
			"retry",
			"monitor",
			"monitor.evaluation_interval",
		} {
			r.SchemaElementOptions.SetEmbeddedObject(f)
		}
	})
}
