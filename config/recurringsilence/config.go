// Package recurringsilence configures the groundcover_recurring_silence resource.
package recurringsilence

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_recurring_silence resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_recurring_silence", func(r *config.Resource) {
		r.ShortGroup = "monitoring"
		r.Kind = "RecurringSilence"
	})
}
