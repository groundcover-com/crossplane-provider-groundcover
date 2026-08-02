# crossplane-provider-groundcover

> **Install** from the Upbound Marketplace — package
> `xpkg.upbound.io/groundcover-com/provider-groundcover`. Or run it from source: see
> [DEVELOPING.md](./DEVELOPING.md).

Manage your [groundcover](https://groundcover.com) resources — **monitors, dashboards,
connected apps, and notification routes** — directly from Kubernetes with
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
# 1. Install the provider (registers the Monitor/Dashboard/ConnectedAppJson/NotificationRoute CRDs)
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
| `groundcover_notification_route` | `kind: NotificationRoute` |
| `groundcover_storage_management_policy` | `kind: StorageManagementPolicy` (adopts the seeded policy; delete only stops managing it) |
| `provider "groundcover" { api_key, backend_id }` | `ProviderConfig` + a credentials `Secret` |

The connected-app `data` is a JSON string here (Crossplane/upjet can't represent the
dynamic-object form Terraform uses). Everything else is the same shape.

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
