package config

import (
	"github.com/crossplane/upjet/pkg/config"

	"github.com/groundcover-com/terraform-provider-groundcover/pkg/tfprovider"

	"github.com/groundcover-com/crossplane-provider-groundcover/config/apikey"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/connectedapp"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/dashboard"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/dataintegration"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/ingestionkey"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/logspipeline"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/metricsaggregation"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/metricspipeline"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/monitor"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/notificationroute"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/policy"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/secret"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/serviceaccount"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/silence"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/synthetictest"
	"github.com/groundcover-com/crossplane-provider-groundcover/config/tracespipeline"
)

const (
	// resourcePrefix is the Terraform provider's resource name prefix.
	resourcePrefix = "groundcover"
	// modulePath is this Crossplane provider module's import path; upjet uses it to
	// generate import statements in the generated API and controller code.
	modulePath = "github.com/groundcover-com/crossplane-provider-groundcover"
	// rootGroup is the API group suffix for all generated CRDs.
	rootGroup = "groundcover.com"
)

// GetProvider builds the upjet provider configuration, shared by the generator
// (cmd/generator) and the runtime provider binary (cmd/provider). It reads the embedded
// Terraform provider schema and coerces dynamic attributes so upjet can ingest it (see
// schema.go).
//
// The provider is a terraform-plugin-framework provider, so only the
// TerraformPluginFramework include list is populated; the SDKv2/CLI include lists stay
// empty. The include list is scoped to the configured resources via ExternalNameConfigured.
func GetProvider() *config.Provider {
	pc := config.NewProvider(
		liftNestedAttributesToBlocks(stripSensitiveInBlocks(coerceDynamicAttributesToString(schemaJSON), "groundcover_synthetic_test")),
		resourcePrefix,
		modulePath,
		nil,
		config.WithRootGroup(rootGroup),
		config.WithShortName(resourcePrefix),
		// This is a terraform-plugin-framework provider, so all resources are sourced
		// through the PF include list. The CLI and SDKv2 include lists default to ".+"
		// (match everything), which would double-register every resource — empty them.
		config.WithIncludeList(nil),
		config.WithTerraformPluginSDKIncludeList(nil),
		config.WithTerraformPluginFrameworkIncludeList(ExternalNameConfigured()),
		// upjet introspects PF resource schemas through the live provider instance.
		config.WithTerraformPluginFrameworkProvider(tfprovider.New("dev")()),
		config.WithDefaultResourceOptions(ExternalNameConfigurations()),
	)

	for _, configure := range []func(*config.Provider){
		monitor.Configure,
		dashboard.Configure,
		connectedapp.Configure,
		notificationroute.Configure,
		apikey.Configure,
		ingestionkey.Configure,
		serviceaccount.Configure,
		policy.Configure,
		secret.Configure,
		silence.Configure,
		synthetictest.Configure,
		dataintegration.Configure,
		logspipeline.Configure,
		metricspipeline.Configure,
		tracespipeline.Configure,
		metricsaggregation.Configure,
	} {
		configure(pc)
	}

	pc.ConfigureResources()
	return pc
}
