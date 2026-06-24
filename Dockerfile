# Runtime image for the groundcover Crossplane provider controller.
#
# The binary is produced by `make provider-binary` (static, linux) into _output/, which is
# this image's build context. `make image` builds it; `make xpkg` embeds it into the
# Crossplane package. Distroless + non-root to match Crossplane's runtime expectations.
FROM gcr.io/distroless/static:nonroot

COPY provider /usr/local/bin/crossplane-provider

USER 65532
ENTRYPOINT ["/usr/local/bin/crossplane-provider"]
