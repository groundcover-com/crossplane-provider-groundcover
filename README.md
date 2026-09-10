# crossplane-provider-groundcover

> **Install** from the Upbound Marketplace — package
> `xpkg.upbound.io/groundcover-com/provider-groundcover`. Or run it from source: see
> [DEVELOPING.md](./DEVELOPING.md).

Manage your [groundcover](https://groundcover.com) resources — **monitors, dashboards,
connected apps, data integrations, and notification routes** — directly from Kubernetes with
[Crossplane](https://crossplane.io), instead of Terraform. You write Kubernetes manifests
(`kind: Monitor`, etc.); Crossplane continuously reconciles them against the groundcover API.

It's generated from the [groundcover Terraform provider](https://github.com/groundcover-com/terraform-provider-groundcover)
with [upjet](https://github.com/crossplane/upjet), so it talks to the exact same API and
reuses the same drift handling — you just drive it the GitOps/Crossplane way.

> **Coming from the groundcover Terraform provider?** Resource shapes mirror the Terraform
> resources. Monitors use the **typed** `groundcover_monitor_v2` schema (the legacy
> YAML-blob `groundcover_monitor` is not exposed). See
> [Coming from Terraform](#coming-from-terraform) for the full mapping.

## Prerequisites

- A Kubernetes cluster with **Crossplane installed** ([install guide](https://docs.crossplane.io/latest/get-started/install/)).
- A groundcover **API key** and **backend id** (Settings → API Keys in the groundcover app).

## Quick start

Runnable manifests live in [`examples/`](./examples). Apply them in order:

```bash
# 1. Install the provider (registers every groundcover CRD: Monitor, Dashboard,
#    ConnectedAppJson, DataIntegration, NotificationRoute, Secret, Policy, APIKey,
#    IngestionKey, ServiceAccount, Skill, SyntheticTest, RecurringSilence, the
#    Logs/Metrics/Traces pipelines, MetricsAggregation and StorageManagementPolicy)
kubectl apply -f examples/provider.yaml
kubectl wait provider/provider-groundcover --for=condition=Healthy --timeout=2m

# 2. Credentials: create the Secret (see the header of providerconfig.yaml), then:
kubectl create secret generic groundcover-creds -n crossplane-system \
  --from-literal=credentials='{"api_key":"<API_KEY>","backend_id":"<BACKEND_ID>"}'
kubectl apply -f examples/providerconfig.yaml

# 3. Create a monitor
kubectl apply -f examples/monitor.yaml
kubectl get monitor gcql-logs-error-count    # SYNCED=True, READY=True once created
```

Edit a manifest and re-apply to update; `kubectl delete` removes the resource from groundcover.

## Examples

| Resource | Example | Notes |
|---|---|---|
| Monitor | [`examples/monitor.yaml`](./examples/monitor.yaml) | typed v2 fields (title, severity, query, threshold, …); `kubectl explain monitor.spec.forProvider` |
| Dashboard | [`examples/dashboard.yaml`](./examples/dashboard.yaml) | `kubectl explain dashboard.spec.forProvider` for the schema |
| ConnectedAppJson | [`examples/connectedappjson.yaml`](./examples/connectedappjson.yaml) | sensitive `data` supplied via a Secret reference |
| DataIntegration | [`AWS`](./examples/dataintegration-aws.yaml), [`PostgreSQL DB monitoring`](./examples/dataintegration-postgresql.yaml), [`ClickHouse DB monitoring`](./examples/dataintegration-clickhouse.yaml) | AWS capability blocks and database health/query-statistics collection; see [Data integrations](#data-integrations) |
| Secret | — | `secrets.groundcover.com` `kind: Secret`, not a core Kubernetes Secret. Produces the `secretRef::store::<id>` references used inside a DataIntegration `config` |
| NotificationRoute | [`examples/notificationroute.yaml`](./examples/notificationroute.yaml) | routes issues to connected apps by status; references a connected-app id |
| Install / config | [`examples/provider.yaml`](./examples/provider.yaml), [`examples/providerconfig.yaml`](./examples/providerconfig.yaml) | |

> `api_url` defaults to `https://api.groundcover.com`; add it to the credentials Secret JSON
> only if your tenant uses a different API host.

## Coming from Terraform

| groundcover Terraform | This provider (Crossplane) |
|---|---|
| `groundcover_monitor_v2` (typed) | `kind: Monitor` (typed `spec.forProvider`) |
| `groundcover_dashboard` | `kind: Dashboard` |
| `groundcover_connected_app` (`data = { ... }`) | `kind: ConnectedAppJson` (`data` as JSON, via `dataSecretRef`) |
| `groundcover_dataintegration` | `kind: DataIntegration` (`config` as JSON string) — see [Data integrations](#data-integrations) |
| `groundcover_notification_route` | `kind: NotificationRoute` |
| `groundcover_storage_management_policy` | `kind: StorageManagementPolicy` (adopts the seeded policy; delete only stops managing it) |
| `groundcover_secret` | `kind: Secret` in `secrets.groundcover.com` |
| `provider "groundcover" { api_key, backend_id }` | `ProviderConfig` + a credentials `Secret` |

The connected-app `data` and data-integration `config` are JSON strings here
(Crossplane/upjet can't represent the dynamic-object form Terraform uses). Everything
else is the same shape.

## Data integrations

`kind: DataIntegration` mirrors Terraform's `groundcover_dataintegration`:
`spec.forProvider.type` picks the data source and `spec.forProvider.config` is the same JSON
the Terraform provider produces with `jsonencode(...)`. The per-type `config` schema — the
supported `type` values, the AWS capability blocks, the emitted metrics — is documented once,
in the [Terraform resource reference][tf-dataintegration], and applies verbatim here: both
providers post the same JSON to the same API.

Runnable manifests: [AWS](./examples/dataintegration-aws.yaml),
[PostgreSQL](./examples/dataintegration-postgresql.yaml),
[ClickHouse](./examples/dataintegration-clickhouse.yaml).

[tf-dataintegration]: https://registry.terraform.io/providers/groundcover-com/groundcover/latest/docs/resources/dataintegration

### Pausing

`config.enabled` is accepted for compatibility, but it is not what stops collection — set
`spec.forProvider.isPaused: true`. The examples set both; `isPaused` is the one this provider
manages.

### Where the integration runs, and secrets

Without `spec.forProvider.cluster` the integration runs in the groundcover backend. Set it to
one of your groundcover cluster names to run it from that cluster's integrations agent
instead — needed when the target is only reachable from inside your network, and when
`config` refers to a Kubernetes Secret.

Credentials inside `config` are `secretRef` strings, never plaintext:

| Form | Resolved against | Needs `cluster` |
|---|---|---|
| `secretRef::store::<id>` | the groundcover secret store. Create the secret with this provider's `kind: Secret` (group `secrets.groundcover.com` — not a core Kubernetes Secret) and use the id it reports | no |
| `secretRef::k8s::<namespace>::<secret-name>::<key>` | a Kubernetes Secret in the cluster running the integration, read by the agent. Create that Secret yourself | **yes** |

A `secretRef::k8s::` reference has no Kubernetes API to read from when the integration runs
in the backend, which is why the two database examples set `cluster` next to it.

### `type` and `cluster` are immutable

Both force replacement in the underlying Terraform resource. Changing either deletes the
integration and creates a new one under a **new external name** (a new id). Emitted metrics
carry that id in `gc_integration_id`, so series from before and after do not join.

### Adopting an existing integration

To manage an integration that already exists in groundcover instead of creating a second one,
set its id as the external name and give the matching `type`:

```yaml
apiVersion: integrations.groundcover.com/v1alpha1
kind: DataIntegration
metadata:
  name: prod-aws
  annotations:
    crossplane.io/external-name: "8f14e45f-ceea-467a-9a1b-2c9b7e0d3f21" # the integration's id
spec:
  providerConfigRef:
    name: default
  forProvider:
    type: aws
    config: | # must match what is configured in groundcover, or the next reconcile updates it
      { ... }
```

This is the Crossplane equivalent of
`terraform import groundcover_dataintegration.example <type>:<id>`: the annotation carries the
`<id>` half, `spec.forProvider.type` the `<type>` half.

### AWS (`type: aws`)

The consolidated AWS integration configures an account once and enables one or more capability
blocks: `vpc`, `dynamodb`, `rds`. AWS account settings are integration-wide — put `regions`,
`roleArn`, `stsRegion` and `scrapeInterval` at the root of `config`, never inside a capability
block. To use different regions, accounts, roles or cadences per capability, create separate
`DataIntegration` resources.

The backend validates `config` and rejects it whole, so a bad manifest surfaces as
`SYNCED=False` rather than a partly applied integration:

- `version` must be `1`.
- Unknown keys are rejected at **every** level, including inside a capability block — and
  `regions`, `roleArn`, `stsRegion` or `scrapeInterval` inside a block is an unknown key.
- `regions` and `scrapeInterval` are required at the root. `scrapeInterval` has an inclusive
  `1m` minimum.
- At least one capability block must be present **and** enabled. An empty object such as
  `"vpc": {}` counts as absent, so always set at least one field inside a block.

Required IAM permissions: `vpc` needs `ec2:DescribeSubnets`; `dynamodb` needs
`dynamodb:ListTables` and `dynamodb:DescribeTable`; `rds` needs `rds:DescribeDBInstances` plus
`logs:GetLogEvents` on the `RDSOSMetrics` log group for Enhanced Monitoring. The
[Terraform reference][tf-dataintegration] lists the metrics and labels each capability emits.

## How drift is handled

No custom drift logic in this repo. upjet runs the groundcover provider's own `Read` on
every reconcile, so the existing suppression (dashboard YAML normalization, connected-app
`data_hash`) applies unchanged — no perpetual diffs.

## Publishing

The provider ships as a Crossplane package (`.xpkg`) destined for the default Crossplane
registry, [`xpkg.crossplane.io`](https://blog.crossplane.io/new-default-crossplane-registry-in-crossplane-1-15/).
The build pipeline is wired up (see [DEVELOPING.md](./DEVELOPING.md#packaging)):

```bash
make xpkg VERSION=v1.16.1          # build the package locally (no push)
make publish ALLOW_PUBLISH=true VERSION=v1.16.1   # push — guarded, intentionally manual
```

`make publish` refuses to run without `ALLOW_PUBLISH=true`: the package is **not published
yet** by deliberate choice — the team tries it out of source / an internal registry first,
and going to the public registry is an explicit, separate decision. Nothing publishes
automatically.

## Status

Resource reconciliation (monitor, dashboard, connected-app-json, notification-route) is
**verified end-to-end** against a live backend, in CI on every change. The package builds
(`make xpkg`) but is **not published** yet — build/run from source for now (see
[DEVELOPING.md](./DEVELOPING.md)).
