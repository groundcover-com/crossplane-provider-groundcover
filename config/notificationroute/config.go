// Package notificationroute configures the groundcover_notification_route resource. Its
// schema is fully typed (no dynamic attributes), so it generates cleanly with no schema
// coercion or controller decorator. The renotification_interval duration is normalized
// in the provider's Read, and that drift suppression carries through upjet.
package notificationroute

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_notification_route resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_notification_route", func(r *config.Resource) {
		r.ShortGroup = "notifications"
		r.Kind = "NotificationRoute"
		// notification_settings is a framework SingleNestedAttribute (an object). The schema
		// lift turns it into an SDKv2 singleton list so upjet can build types; collapse it
		// back to an embedded object so the generated CRD field — and the params map fed to
		// the framework provider at runtime — is object-shaped, matching the provider schema.
		// SetEmbeddedObject only changes type generation; it does NOT register a runtime
		// singleton-list conversion (AddSingletonListConversion would, which is wrong here
		// because the framework provider wants an object, not a list).
		if r.SchemaElementOptions == nil {
			r.SchemaElementOptions = config.SchemaElementOptions{}
		}
		r.SchemaElementOptions.SetEmbeddedObject("notification_settings")
	})
}
