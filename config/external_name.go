// Package config holds the upjet provider configuration that drives CRD/controller
// generation for the groundcover Crossplane provider.
package config

import (
	"context"

	"github.com/crossplane/upjet/pkg/config"
	"github.com/pkg/errors"
)

// storageManagementPolicyExternalName identifies the policy by data_type: the resource
// has no server-assigned "id" attribute — a policy always exists per data type and
// data_type is the Terraform import ID. The empty-external-name guards keep the first
// reconcile (before adoption sets the external name) from clobbering spec's data_type.
var storageManagementPolicyExternalName = config.ExternalName{
	SetIdentifierArgumentFn: func(base map[string]any, externalName string) {
		if externalName != "" {
			base["data_type"] = externalName
		}
	},
	GetExternalNameFn: func(tfstate map[string]any) (string, error) {
		if dt, ok := tfstate["data_type"].(string); ok && dt != "" {
			return dt, nil
		}
		return "", errors.New("data_type not found in tfstate")
	},
	GetIDFn: func(_ context.Context, externalName string, parameters map[string]any, _ map[string]any) (string, error) {
		if externalName != "" {
			return externalName, nil
		}
		dt, _ := parameters["data_type"].(string)
		return dt, nil
	},
	DisableNameInitializer: true,
}

var dataIntegrationExternalName = config.NewExternalNameFrom(
	config.IdentifierFromProvider,
	config.WithGetIDFn(func(fn config.GetIDFn, ctx context.Context, externalName string, parameters map[string]any, terraformProviderConfig map[string]any) (string, error) {
		if externalName != "" {
			return fn(ctx, externalName, parameters, terraformProviderConfig)
		}
		integrationType, _ := parameters["type"].(string)
		if integrationType == "" {
			integrationType = "aws"
		}
		return integrationType + ":00000000-0000-0000-0000-000000000000", nil
	}),
)

// ExternalNameConfigs maps each Terraform resource to its external-name handling.
// groundcover resources are identified by a server-assigned UUID returned in the
// Terraform "id" field, so they all use IdentifierFromProvider: the external name is
// whatever the provider assigns on create, and no name field is sent on the request.
// The storage management policy is the exception — see storageManagementPolicyExternalName.
var ExternalNameConfigs = map[string]config.ExternalName{
	"groundcover_monitor_v2_json":    config.IdentifierFromProvider,
	"groundcover_dashboard":          config.IdentifierFromProvider,
	"groundcover_connected_app_json": config.IdentifierFromProvider,
	"groundcover_notification_route": config.IdentifierFromProvider,
	"groundcover_apikey":             config.IdentifierFromProvider,
	"groundcover_ingestionkey":       config.IdentifierFromProvider,
	"groundcover_serviceaccount":     config.IdentifierFromProvider,
	"groundcover_policy":             config.IdentifierFromProvider,
	"groundcover_secret":             config.IdentifierFromProvider,
	"groundcover_recurring_silence":  config.IdentifierFromProvider,
	"groundcover_skill":              config.IdentifierFromProvider,
	"groundcover_synthetic_test":     config.IdentifierFromProvider,
	"groundcover_dataintegration":    dataIntegrationExternalName,
	"groundcover_logspipeline":       config.IdentifierFromProvider,
	"groundcover_metricspipeline":    config.IdentifierFromProvider,
	"groundcover_tracespipeline":     config.IdentifierFromProvider,
	"groundcover_metricsaggregation": config.IdentifierFromProvider,

	"groundcover_storage_management_policy": storageManagementPolicyExternalName,
}

// ExternalNameConfigured returns the list of Terraform resources that have an
// external-name configuration, in the regex form upjet's include lists expect (anchored
// with a trailing "$"). It feeds WithTerraformPluginFrameworkIncludeList so only the
// configured resources are generated.
func ExternalNameConfigured() []string {
	l := make([]string, 0, len(ExternalNameConfigs))
	for name := range ExternalNameConfigs {
		l = append(l, name+"$")
	}
	return l
}

// ExternalNameConfigurations applies the ExternalNameConfigs to the matching resource
// during configuration. Registered as a default resource option.
func ExternalNameConfigurations() config.ResourceOption {
	return func(r *config.Resource) {
		if e, ok := ExternalNameConfigs[r.Name]; ok {
			r.ExternalName = e
		}
	}
}
