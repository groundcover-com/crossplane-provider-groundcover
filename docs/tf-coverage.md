# Terraform provider coverage

Which `terraform-provider-groundcover` resources the Crossplane provider exposes.
Source of truth for what's configured: `config/*/config.go` (each calls
`AddResourceConfigurator("groundcover_<name>")`).

| # | TF registry resource | Crossplane | Notes |
|---|---|---|---|
| 1 | `groundcover_apikey` | ✅ | `rbac.APIKey` |
| 2 | `groundcover_connected_app` | ⚪️ by design | covered by `_json` variant |
| 3 | `groundcover_connected_app_json` | ✅ | `integrations.ConnectedAppJson` |
| 4 | `groundcover_dashboard` | ✅ | `dashboards.Dashboard` |
| 5 | `groundcover_dataintegration` | ✅ | `integrations.DataIntegration` |
| 6 | `groundcover_ingestionkey` | ✅ | `rbac.IngestionKey` |
| 7 | `groundcover_logspipeline` | ✅ | `pipelines.LogsPipeline` |
| 8 | `groundcover_metricsaggregation` | ✅ | `pipelines.MetricsAggregation` |
| 9 | `groundcover_metricspipeline` | ✅ | `pipelines.MetricsPipeline` |
| 10 | `groundcover_monitor` | 🚫 excluded | deliberately not supported |
| 11 | `groundcover_monitor_v2` | ⚪️ by design | covered by `_json` variant |
| 12 | `groundcover_monitor_v2_json` | ✅ | `monitoring.Monitor` |
| 13 | `groundcover_notification_route` | ✅ | `notifications.NotificationRoute` |
| 14 | `groundcover_policy` | ✅ | `rbac.Policy` |
| 15 | `groundcover_secret` | ✅ | `secrets.Secret` |
| 16 | `groundcover_serviceaccount` | ✅ | `rbac.ServiceAccount` |
| 17 | `groundcover_silence` | ✅ | `monitoring.Silence` |
| 18 | `groundcover_synthetic_test` | ❌ **gap** | not configured, no `_json` alternative |
| 19 | `groundcover_tracespipeline` | ✅ | `pipelines.TracesPipeline` |

Legend: ✅ supported · ⚪️ covered by `_json` variant · 🚫 deliberately excluded · ❌ genuine gap

**Summary:** 15/19 supported. The only genuine gap is `groundcover_synthetic_test`.
`connected_app` and `monitor_v2` are intentionally served by their `_json` twins;
`monitor` is a deliberate exclusion.

## Adding a resource

Same three-file recipe used for every resource here (see `config/policy/` as the
reference):

1. `config/<name>/config.go` — `AddResourceConfigurator`, set `ShortGroup` + `Kind`.
2. `config/external_name.go` — add the `groundcover_<name>` external-name entry
   (`IdentifierFromProvider` for server-assigned UUIDs).
3. `config/provider.go` — import the package and add `<name>.Configure` to the list.
4. `make generate crds`.

**Watch for nested-object fields.** `liftNestedAttributesToBlocks` (config/schema.go)
turns terraform-plugin-framework `SingleNestedAttribute`s into singleton-list blocks.
Any such field must be registered with `SchemaElementOptions.SetEmbeddedObject(path)`
(dotted TF path) or create fails at runtime with
`<field>: invalid JSON, expected "{", got "["`. Lists (`ListNestedAttribute`) stay
lists — don't embed those. Grep the resource's schema in the TF provider before
generating.
