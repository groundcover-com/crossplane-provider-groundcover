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

---

## Prerequisites

- A Kubernetes cluster with **Crossplane installed** ([install guide](https://docs.crossplane.io/latest/software/install/)).
- A groundcover **API key** and **backend id** (Settings → API Keys in the groundcover app).

## 1. Install the provider

```yaml
apiVersion: pkg.crossplane.io/v1
kind: Provider
metadata:
  name: provider-groundcover
spec:
  package: ghcr.io/groundcover-com/crossplane-provider-groundcover:v0.1.0  # ⚠️ image not published yet — see Status
```

```bash
kubectl apply -f provider.yaml
kubectl wait provider/provider-groundcover --for=condition=Healthy --timeout=2m
```

Installing the provider registers the groundcover CRDs: `Monitor`, `Dashboard`, `ConnectedAppJson`.

## 2. Configure credentials

Put your groundcover credentials in a `Secret` as a JSON object, then point a
`ProviderConfig` at it. `api_url` defaults to `https://api.groundcover.com` — set it only
if your tenant uses a different API host.

```bash
kubectl create secret generic groundcover-creds -n crossplane-system \
  --from-literal=credentials='{"api_key":"<YOUR_API_KEY>","backend_id":"<YOUR_BACKEND_ID>"}'
```

```yaml
apiVersion: groundcover.com/v1beta1
kind: ProviderConfig
metadata:
  name: default
spec:
  credentials:
    source: Secret
    secretRef:
      namespace: crossplane-system
      name: groundcover-creds
      key: credentials
```

## 3. Create a monitor

```yaml
apiVersion: monitoring.groundcover.com/v1alpha1
kind: Monitor
metadata:
  name: pod-crash-looping
spec:
  providerConfigRef:
    name: default
  forProvider:
    # Same YAML you'd put in groundcover_monitor.monitor_yaml in Terraform.
    monitorYaml: |
      title: K8s Pod Crash Looping
      display:
        header: K8s Pod Crash Looping
      severity: S2
      measurementType: state
      model:
        queries:
        - name: q1
          dataType: metrics
          pipeline:
            metric: groundcover_kube_pod_container_status_waiting_reason
        thresholds:
        - name: t1
          inputName: q1
          operator: gt
          values:
          - 0
      evaluationInterval:
        interval: 1m
        pendingFor: 5m
```

```bash
kubectl apply -f monitor.yaml
kubectl get monitor pod-crash-looping       # SYNCED=True, READY=True once created
```

Edit the manifest and re-apply to update; `kubectl delete` removes it from groundcover.

## 4. Other resources

**Dashboard** (`dashboards.groundcover.com/v1alpha1`) — `spec.forProvider` fields: `name`,
`preset` (the dashboard JSON), `team`, `description`, `override`. Run
`kubectl explain dashboard.spec.forProvider` for the full schema.

**ConnectedAppJson** (`integrations.groundcover.com/v1alpha1`) — the connected-app `data`
is sensitive, so it's supplied via a Secret reference, not inline:

```bash
kubectl create secret generic slack-app-data -n crossplane-system \
  --from-literal=data='{"url":"https://hooks.slack.com/services/XXX/YYY/ZZZ"}'
```

```yaml
apiVersion: integrations.groundcover.com/v1alpha1
kind: ConnectedAppJson
metadata:
  name: alerts-slack
spec:
  providerConfigRef:
    name: default
  forProvider:
    name: alerts-slack
    type: slack-webhook            # slack-webhook, pagerduty, opsgenie, incidentio, webhook, rootly, ms-teams
    dataSecretRef:
      namespace: crossplane-system
      name: slack-app-data
      key: data                    # JSON object matching the app type
```

## Coming from Terraform

| groundcover Terraform | This provider (Crossplane) |
|---|---|
| `groundcover_monitor` (`monitor_yaml`) | `kind: Monitor` (`spec.forProvider.monitorYaml`) |
| `groundcover_dashboard` | `kind: Dashboard` |
| `groundcover_connected_app` (`data = { ... }`) | `kind: ConnectedAppJson` (`data` as JSON, via `dataSecretRef`) |
| `provider "groundcover" { api_key, backend_id }` | `ProviderConfig` + a credentials `Secret` |

The connected-app `data` is a JSON string here (Crossplane/upjet can't represent the
dynamic-object form `groundcover_connected_app` uses in Terraform). Everything else is the
same shape.

## How drift is handled

There's no custom drift logic in this repo. upjet runs the groundcover provider's own
`Read` on every reconcile, so the existing suppression (monitor/dashboard YAML
normalization, connected-app `data_hash`) applies unchanged — no perpetual diffs.

## Status

POC (BE-2207). The provider package is **not published yet**, so the install image above is
a placeholder. Until release, build from source — see [DEVELOPING.md](./DEVELOPING.md) for
`make schema` / `make generate` / `make build` and the TF-provider version pin.
