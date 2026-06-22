# crossplane-provider-groundcover

> ⚠️ **PRIVATE / INTERNAL — do not make this repository public.** POC under BE-2207.
> No release/publish automation is configured yet. Flip to public only on an explicit
> go-public decision.

Manage your [groundcover](https://groundcover.com) resources — **monitors, dashboards, and
connected apps** — directly from Kubernetes with [Crossplane](https://crossplane.io),
instead of Terraform. You write Kubernetes manifests (`kind: Monitor`, etc.); Crossplane
continuously reconciles them against the groundcover API.

It's generated from the [groundcover Terraform provider](https://github.com/groundcover-com/terraform-provider-groundcover)
with [upjet](https://github.com/crossplane/upjet), so it talks to the exact same API and
reuses the same drift handling — you just drive it the GitOps/Crossplane way.

> **Already on the groundcover Terraform provider?** Your `monitor_yaml` carries over
> verbatim. See [Coming from Terraform](#coming-from-terraform).

## Prerequisites

- A Kubernetes cluster with **Crossplane installed** ([install guide](https://docs.crossplane.io/latest/software/install/)).
- A groundcover **API key** and **backend id** (Settings → API Keys in the groundcover app).

## Quick start

Runnable manifests live in [`examples/`](./examples). Apply them in order:

```bash
# 1. Install the provider (registers the Monitor/Dashboard/ConnectedAppJson CRDs)
kubectl apply -f examples/provider.yaml
kubectl wait provider/provider-groundcover --for=condition=Healthy --timeout=2m

# 2. Credentials: create the Secret (see the header of providerconfig.yaml), then:
kubectl create secret generic groundcover-creds -n crossplane-system \
  --from-literal=credentials='{"api_key":"<API_KEY>","backend_id":"<BACKEND_ID>"}'
kubectl apply -f examples/providerconfig.yaml

# 3. Create a monitor
kubectl apply -f examples/monitor.yaml
kubectl get monitor pod-crash-looping        # SYNCED=True, READY=True once created
```

Edit a manifest and re-apply to update; `kubectl delete` removes the resource from groundcover.

## Examples

| Resource | Example | Notes |
|---|---|---|
| Monitor | [`examples/monitor.yaml`](./examples/monitor.yaml) | `monitorYaml` = the same YAML as `groundcover_monitor.monitor_yaml` |
| Dashboard | [`examples/dashboard.yaml`](./examples/dashboard.yaml) | `kubectl explain dashboard.spec.forProvider` for the schema |
| ConnectedAppJson | [`examples/connectedappjson.yaml`](./examples/connectedappjson.yaml) | sensitive `data` supplied via a Secret reference |
| Install / config | [`examples/provider.yaml`](./examples/provider.yaml), [`examples/providerconfig.yaml`](./examples/providerconfig.yaml) | |

> `api_url` defaults to `https://api.groundcover.com`; add it to the credentials Secret JSON
> only if your tenant uses a different API host.

## Coming from Terraform

| groundcover Terraform | This provider (Crossplane) |
|---|---|
| `groundcover_monitor` (`monitor_yaml`) | `kind: Monitor` (`spec.forProvider.monitorYaml`) |
| `groundcover_dashboard` | `kind: Dashboard` |
| `groundcover_connected_app` (`data = { ... }`) | `kind: ConnectedAppJson` (`data` as JSON, via `dataSecretRef`) |
| `provider "groundcover" { api_key, backend_id }` | `ProviderConfig` + a credentials `Secret` |

The connected-app `data` is a JSON string here (Crossplane/upjet can't represent the
dynamic-object form Terraform uses). Everything else is the same shape.

## How drift is handled

No custom drift logic in this repo. upjet runs the groundcover provider's own `Read` on
every reconcile, so the existing suppression (monitor/dashboard YAML normalization,
connected-app `data_hash`) applies unchanged — no perpetual diffs.

## Status

POC (BE-2207). The provider package is **not published yet**, so `examples/provider.yaml`
points at a placeholder image and the manifests are unverified end-to-end pending the
in-flight PRs. Build from source meanwhile — see [DEVELOPING.md](./DEVELOPING.md).
