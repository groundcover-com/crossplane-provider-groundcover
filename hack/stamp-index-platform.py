#!/usr/bin/env python3
"""Stamp the platform descriptor onto an xpkg's package manifest in the registry.

`up alpha xpkg append` converts a single-arch package image into an OCI image index
to attach marketplace extensions (icon/readme), but it does NOT carry the platform
(architecture/os) onto the package manifest's index descriptor. Upbound's marketplace
listing generator appears to need that platform descriptor (Upbound Official providers
like provider-databricks have it), so without it the listing URL/icon don't render.

This re-reads the published index, copies architecture/os from the package image's
config onto its index descriptor, and PUTs the index back to the same tag. Idempotent.

Usage: UP_TOKEN=<upbound token> stamp-index-platform.py <registry> <repo> <tag>
  e.g. stamp-index-platform.py xpkg.upbound.io groundcover-com/provider-groundcover v0.1.0
"""
import json
import os
import sys
import urllib.request
import urllib.error

OCI_INDEX = "application/vnd.oci.image.index.v1+json"
DOCKER_LIST = "application/vnd.docker.distribution.manifest.list.v2+json"
DOCKER_MANIFEST = "application/vnd.docker.distribution.manifest.v2+json"
OCI_MANIFEST = "application/vnd.oci.image.manifest.v1+json"
EXT_ANNOTATION = "io.crossplane.xpkg"  # value "xpkg-extensions" marks the extensions manifest


def req(url, token, accept=None, data=None, method=None, content_type=None):
    headers = {"Authorization": f"Bearer {token}"}
    if accept:
        headers["Accept"] = accept
    if content_type:
        headers["Content-Type"] = content_type
    r = urllib.request.Request(url, headers=headers, data=data, method=method)
    return urllib.request.urlopen(r)


def main():
    reg, name, tag = sys.argv[1], sys.argv[2], sys.argv[3]
    up_token = os.environ["UP_TOKEN"]

    # 1. exchange the Upbound token for a registry bearer with pull+push scope
    scope = f"repository:{name}:pull,push"
    tok_url = f"https://{reg}/service/token?scope={scope}&service={reg}"
    token = json.load(req(tok_url, up_token))["token"]
    # fail fast if push wasn't granted (e.g. token lacks write access)
    import base64
    payload = token.split(".")[1]
    payload += "=" * (-len(payload) % 4)
    actions = json.loads(base64.urlsafe_b64decode(payload)).get("access", [{}])[0].get("actions", [])
    if "push" not in actions:
        sys.exit(f"registry token lacks push scope (got {actions}); token needs write access to {name}")

    base = f"https://{reg}/v2/{name}"
    # 2. fetch the index
    idx_accept = f"{OCI_INDEX}, {DOCKER_LIST}"
    idx = json.load(req(f"{base}/manifests/{tag}", token, accept=idx_accept))
    if "manifests" not in idx:
        sys.exit(f"{tag} is not a multi-manifest index (mediaType {idx.get('mediaType')}); nothing to stamp")

    # 3. the package manifest is the one that is NOT the extensions manifest
    pkgs = [m for m in idx["manifests"]
            if m.get("annotations", {}).get(EXT_ANNOTATION) != "xpkg-extensions"]
    changed = False
    for pkg in pkgs:
        if pkg.get("platform", {}).get("architecture") and pkg.get("platform", {}).get("os"):
            continue  # already stamped
        # 4. read architecture/os from the package image config
        man = json.load(req(f"{base}/manifests/{pkg['digest']}", token,
                            accept=f"{DOCKER_MANIFEST}, {OCI_MANIFEST}"))
        cfg = json.load(req(f"{base}/blobs/{man['config']['digest']}", token))
        arch, os_ = cfg.get("architecture"), cfg.get("os")
        if not arch or not os_:
            sys.exit(f"package config missing architecture/os ({arch}/{os_})")
        pkg["platform"] = {"architecture": arch, "os": os_}
        changed = True
        print(f"stamped {pkg['digest'][:19]} -> {arch}/{os_}")

    if not changed:
        print("platform already present; nothing to do")
        return

    # 5. PUT the index back to the same tag
    body = json.dumps(idx).encode()
    ct = idx.get("mediaType", OCI_INDEX)
    resp = req(f"{base}/manifests/{tag}", token, data=body, method="PUT", content_type=ct)
    print(f"PUT {tag}: HTTP {resp.status}, digest {resp.headers.get('Docker-Content-Digest')}")


if __name__ == "__main__":
    try:
        main()
    except urllib.error.HTTPError as e:
        sys.exit(f"HTTP {e.code} {e.reason}: {e.read().decode(errors='replace')[:300]}")
