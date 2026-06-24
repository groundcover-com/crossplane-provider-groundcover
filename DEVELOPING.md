# Developing crossplane-provider-groundcover

This is an [upjet](https://github.com/crossplane/upjet)-generated Crossplane provider. It
holds no business logic — `config/` selects which groundcover resources to generate,
`internal/clients` maps credentials to the provider config, and everything under
`apis/<group>` and `internal/controller/<group>` is generated from the groundcover
Terraform provider's schema.

## Build / regenerate

```bash
make schema      # config/schema.json from the groundcover Terraform provider
make generate    # upjet pipeline + controller-gen (deepcopy) + angryjet (managed method sets)
make build
```

`make generate` needs a schema that includes `groundcover_connected_app_json`, i.e. a TF
provider build that ships it. Until that's released, generate the schema from a local
provider build (the `replace` below points at a sibling checkout).

## The one version pin that matters

This provider **embeds** the groundcover Terraform provider it's generated from, and runs
its `Read` at reconcile time. Pin it to a version that has `data_hash`, duration
normalization, and `groundcover_connected_app_json` — build against an older one and the
provider ships the same drift it was meant to fix.

- **Dev:** `go.mod` has `replace github.com/groundcover-com/terraform-provider-groundcover => ../terraform-provider-groundcover` for local co-development.
- **Release:** drop the `replace`, require the published version, regenerate, tag.

When a new TF provider release changes resource schema or `Read`/drift behavior, bump the
dependency, `make generate`, and cut a new provider release.

## Layout

| Path | What |
|---|---|
| `config/` | upjet provider config: include list, external-name, per-resource settings |
| `internal/clients/` | `SetupFn` — maps `ProviderConfig` credentials to the groundcover provider |
| `apis/<group>/` | generated CRD API types (gitignored; `make generate`) |
| `internal/controller/<group>/` | generated controllers (gitignored; `make generate`) |
| `cmd/provider/` | provider binary entrypoint |
| `package/crossplane.yaml` | package metadata (tracked); `package/crds/` is generated |
| `Dockerfile` | controller runtime image (distroless, non-root) |

## Packaging

The provider ships as a Crossplane package (`.xpkg`) for [`xpkg.crossplane.io`](https://blog.crossplane.io/new-default-crossplane-registry-in-crossplane-1-15/).
Needs the [`crossplane` CLI](https://docs.crossplane.io/latest/cli/) and Docker.

```bash
make crds              # generate CRDs into package/crds
make image             # build the controller runtime image
make xpkg VERSION=v1.16.1    # build _output/<provider>-v1.16.1.xpkg (crds + meta + embedded image). No push.
```

`make xpkg` chains `crds` + `image`. Override `REGISTRY`/`ORG`/`PROVIDER_NAME`/`PLATFORM` as
needed (defaults: `xpkg.crossplane.io` / `groundcover-com` / `provider-groundcover` / `linux/amd64`).

Publishing is guarded — it never runs without an explicit opt-in:

```bash
make publish ALLOW_PUBLISH=true VERSION=v1.16.1   # crossplane xpkg push to $REGISTRY/$ORG/...
```

For team testing before going public, push to an internal/private OCI registry instead
(`make publish ALLOW_PUBLISH=true REGISTRY=<internal-registry> VERSION=...`) and point
`examples/provider.yaml` at that image.
