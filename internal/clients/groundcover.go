/*
Copyright 2024 groundcover.
*/

package clients

import (
	"context"
	"encoding/json"

	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/upjet/pkg/terraform"

	"github.com/groundcover-com/terraform-provider-groundcover/pkg/tfprovider"

	"github.com/groundcover-com/crossplane-provider-groundcover/apis/v1beta1"
)

const (
	errNoProviderConfig     = "no providerConfigRef provided"
	errGetProviderConfig    = "cannot get referenced ProviderConfig"
	errTrackUsage           = "cannot track ProviderConfig usage"
	errExtractCredentials   = "cannot extract credentials"
	errUnmarshalCredentials = "cannot unmarshal groundcover credentials as JSON"
)

// TerraformSetupBuilder returns a terraform.SetupFn for the groundcover provider.
//
// Unlike a CLI/SDKv2 provider, this is a terraform-plugin-framework provider: upjet's
// PF connector configures the provider in-process, so the Setup must carry both the
// provider Configuration (api_key/api_url/backend_id) and a live FrameworkProvider
// instance (the connector errors if it is nil). The Requirement/Version fields used by
// the Terraform CLI runner are irrelevant here and left empty.
//
// Credentials are read from the ProviderConfig's referenced secret, expected to be a
// JSON object with keys api_key, api_url, backend_id (matching the provider schema).
func TerraformSetupBuilder(version string) terraform.SetupFn {
	return func(ctx context.Context, c client.Client, mg resource.Managed) (terraform.Setup, error) {
		ps := terraform.Setup{
			// The in-process PF provider; required by upjet's PF connector.
			FrameworkProvider: tfprovider.New(version)(),
		}

		configRef := mg.GetProviderConfigReference()
		if configRef == nil {
			return ps, errors.New(errNoProviderConfig)
		}
		pc := &v1beta1.ProviderConfig{}
		if err := c.Get(ctx, types.NamespacedName{Name: configRef.Name}, pc); err != nil {
			return ps, errors.Wrap(err, errGetProviderConfig)
		}

		t := resource.NewProviderConfigUsageTracker(c, &v1beta1.ProviderConfigUsage{})
		if err := t.Track(ctx, mg); err != nil {
			return ps, errors.Wrap(err, errTrackUsage)
		}

		data, err := resource.CommonCredentialExtractor(ctx, pc.Spec.Credentials.Source, c, pc.Spec.Credentials.CommonCredentialSelectors)
		if err != nil {
			return ps, errors.Wrap(err, errExtractCredentials)
		}
		creds := map[string]string{}
		if err := json.Unmarshal(data, &creds); err != nil {
			return ps, errors.Wrap(err, errUnmarshalCredentials)
		}

		// Map credentials to the groundcover provider's configuration block. Only set
		// keys that were provided so the provider's own defaults/env fallbacks apply.
		cfg := map[string]any{}
		for _, k := range []string{"api_key", "api_url", "backend_id", "org_name"} {
			if v, ok := creds[k]; ok && v != "" {
				cfg[k] = v
			}
		}
		ps.Configuration = cfg
		return ps, nil
	}
}
