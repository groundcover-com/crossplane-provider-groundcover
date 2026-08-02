// Package storagemanagementpolicy configures the groundcover_storage_management_policy resource.
package storagemanagementpolicy

import "github.com/crossplane/upjet/pkg/config"

// Configure registers the groundcover_storage_management_policy resource configuration.
func Configure(p *config.Provider) {
	p.AddResourceConfigurator("groundcover_storage_management_policy", func(r *config.Resource) {
		r.ShortGroup = "storage"
		r.Kind = "StorageManagementPolicy"
	})
}
