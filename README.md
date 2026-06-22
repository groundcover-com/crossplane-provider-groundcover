# provider-groundcover

A [Crossplane](https://crossplane.io) provider for [groundcover](https://groundcover.com),
generated from the [groundcover Terraform provider](https://github.com/groundcover-com/terraform-provider-groundcover)
with [upjet](https://github.com/crossplane/upjet).

This is the **only** thing groundcover-on-Crossplane users need — install this provider and
manage groundcover resources as native Kubernetes CRDs (`Monitor`, `Dashboard`,
`ConnectedAppJson`). It is a separate repo from the Terraform provider on purpose (it
*consumes* the published TF provider); the two are not mixed. Mirrors the
`upbound/provider-datadog` model.

## Why no custom drift logic

upjet runs the groundcover provider's own `Read` at reconcile time, so the provider's
existing drift suppression (YAML normalization for monitors/dashboards, `data_hash` for
connected apps) is reused as-is. No decorator, no re-implementation. The only requirement
is that this provider is built against a TF-provider version that contains those fixes
(see the pin below).

## Resources

| CRD | Terraform resource | Notes |
|-----|--------------------|-------|
| `Monitor` | `groundcover_monitor` | YAML string body |
| `Dashboard` | `groundcover_dashboard` | YAML string body |
| `ConnectedAppJson` | `groundcover_connected_app_json` | `data` as a JSON string (the dynamic `groundcover_connected_app` is not codegen-able by upjet) |

## Build / regenerate

```bash
make schema      # produce config/schema.json from the published groundcover provider
make generate    # upjet pipeline + controller-gen (deepcopy) + angryjet (managed methodsets)
make build
```

`make generate` requires a schema that includes `groundcover_connected_app_json`, i.e. a
TF-provider release that ships it. Until that release exists, point `make schema` at a
local build, or drop a `config/schema.json` generated from the local provider.

## Versioning / the one pin that matters

This provider embeds the TF provider it's generated from. Pin it to a version that has:
`data_hash` + duration normalization + `groundcover_connected_app_json`. Build against an
older version and it ships the same drift it was meant to fix.

- Dev: `go.mod` has `replace github.com/groundcover-com/terraform-provider-groundcover => ../terraform-provider-groundcover` for local co-development.
- Release: drop the `replace`, require the published version.

## Status

POC scaffolding from BE-2207. Generated tree (`apis/<group>`, `internal/controller/<group>`)
is gitignored and reproduced by `make generate`; releases ship the built provider image.
