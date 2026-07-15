package skill

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_skill resource as the Skill kind.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_skill", func(r *config.Resource) {
		r.ShortGroup = "agent"
		r.Kind = "Skill"
	})
}
