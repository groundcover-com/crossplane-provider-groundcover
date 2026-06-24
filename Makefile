# Makefile for the groundcover Crossplane provider (upjet-generated).
#
# This provider is generated from the groundcover Terraform provider that lives in the
# parent directory. See README.md for the full generation runbook and the design notes
# in the BE-2055 spec.

# ====================================================================================
# Terraform provider source (POC: pin to a published groundcover provider version).
# To generate against the in-repo provider instead, build it locally and point
# PROVIDER_SCHEMA at the schema produced by `terraform providers schema -json`.

TERRAFORM_PROVIDER_SOURCE        ?= groundcover-com/groundcover
TERRAFORM_PROVIDER_REPO          ?= https://github.com/groundcover-com/terraform-provider-groundcover
TERRAFORM_PROVIDER_VERSION       ?= 1.14.1
TERRAFORM_PROVIDER_DOWNLOAD_NAME ?= terraform-provider-groundcover

PROVIDER_SCHEMA ?= config/schema.json

GOIMPORTS ?= go run golang.org/x/tools/cmd/goimports@latest

# controller-gen produces deepcopy methods; angryjet produces the crossplane Managed /
# ManagedList method sets. The upjet pipeline emits neither. angryjet is pinned to the
# last commit that targets crossplane-runtime v1 (matching our upjet v1.11.x stack) while
# still using a Go 1.26-compatible go/packages loader; newer commits switched to the
# crossplane-runtime/v2 module and silently generate nothing for our v1 types.
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.5
ANGRYJET ?= go run github.com/crossplane/crossplane-tools/cmd/angryjet@9102d33d29103586ac05524a3e6c209b4e63fe3c

# ====================================================================================
# Targets

.PHONY: help
help: ## Show this help.
	@grep -hE '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "} {printf "  %-18s %s\n", $$1, $$2}'

.PHONY: test
test: ## Run unit tests (observe decorator + strategies).
	go test ./...

.PHONY: build
build: ## Build all non-generated packages.
	go build ./...

.PHONY: schema
schema: ## Produce config/schema.json from the Terraform provider (requires terraform CLI).
	@command -v terraform >/dev/null 2>&1 || { echo "terraform CLI is required for 'make schema'"; exit 1; }
	@echo ">> generating $(PROVIDER_SCHEMA) from $(TERRAFORM_PROVIDER_SOURCE) $(TERRAFORM_PROVIDER_VERSION)"
	@mkdir -p .cache/schema
	@printf 'terraform {\n  required_providers {\n    groundcover = {\n      source  = "%s"\n      version = "%s"\n    }\n  }\n}\n' \
		"$(TERRAFORM_PROVIDER_SOURCE)" "$(TERRAFORM_PROVIDER_VERSION)" > .cache/schema/main.tf
	cd .cache/schema && terraform init -upgrade >/dev/null && terraform providers schema -json > ../../$(PROVIDER_SCHEMA)
	@echo ">> wrote $(PROVIDER_SCHEMA)"

.PHONY: generate
generate: $(PROVIDER_SCHEMA) ## Run the upjet generation pipeline (CRDs, controllers, examples).
	@echo ">> installing goimports (upjet's pipeline shells out to the goimports binary on PATH)"
	go install golang.org/x/tools/cmd/goimports@latest
	@echo ">> running upjet generation pipeline"
	go run ./cmd/generator
	@echo ">> generating deepcopy methods (controller-gen)"
	$(CONTROLLER_GEN) object:headerFile=hack/boilerplate.go.txt paths=./apis/...
	@echo ">> generating crossplane managed method sets (angryjet)"
	$(ANGRYJET) generate-methodsets --header-file=hack/boilerplate.go.txt ./apis/...
	@echo ">> formatting generated code"
	$(GOIMPORTS) -w ./apis ./internal
	go build ./...

$(PROVIDER_SCHEMA):
	@echo "ERROR: $(PROVIDER_SCHEMA) not found. Run 'make schema' first (needs the terraform CLI)."
	@exit 1

# ====================================================================================
# Packaging / publishing (Crossplane registry: xpkg.crossplane.io)
#
# A Crossplane provider ships as an OCI package (.xpkg) that bundles the package metadata
# (package/crossplane.yaml) + CRDs and embeds the controller runtime image. Build it with
# the crossplane CLI: https://docs.crossplane.io/latest/cli/ (`crossplane` on PATH).
#
#   make xpkg                 # build the .xpkg locally (no push)
#   make publish ALLOW_PUBLISH=true VERSION=v1.16.1
#
# Nothing is pushed unless ALLOW_PUBLISH=true is set explicitly — the provider is private
# and unverified end-to-end. See README "Publishing".

REGISTRY      ?= xpkg.crossplane.io
ORG           ?= groundcover-com
PROVIDER_NAME ?= provider-groundcover
VERSION       ?= v0.0.0-dev
PLATFORM      ?= linux/amd64

CONTROLLER_IMAGE ?= $(PROVIDER_NAME)-controller:$(VERSION)
XPKG_REF         ?= $(REGISTRY)/$(ORG)/$(PROVIDER_NAME):$(VERSION)
XPKG_FILE        ?= _output/$(PROVIDER_NAME)-$(VERSION).xpkg
CROSSPLANE       ?= crossplane

.PHONY: crds
crds: generate ## Generate CRDs into package/crds for packaging.
	@mkdir -p package/crds
	$(CONTROLLER_GEN) crd:allowDangerousTypes=true paths=./apis/... output:crd:dir=package/crds

.PHONY: provider-binary
provider-binary: ## Build the static linux controller binary into _output/.
	@mkdir -p _output
	CGO_ENABLED=0 GOOS=$(word 1,$(subst /, ,$(PLATFORM))) GOARCH=$(word 2,$(subst /, ,$(PLATFORM))) \
		go build -o _output/provider ./cmd/provider

.PHONY: image
image: provider-binary ## Build the controller runtime OCI image.
	docker build --platform=$(PLATFORM) -t $(CONTROLLER_IMAGE) -f Dockerfile _output

.PHONY: xpkg
xpkg: crds image ## Build the Crossplane provider package (.xpkg). No push.
	@command -v $(CROSSPLANE) >/dev/null 2>&1 || { echo "crossplane CLI required: https://docs.crossplane.io/latest/cli/"; exit 1; }
	@mkdir -p _output
	$(CROSSPLANE) xpkg build --package-root=package --embed-runtime-image=$(CONTROLLER_IMAGE) --package-file=$(XPKG_FILE)
	@echo ">> built $(XPKG_FILE)"

.PHONY: publish
publish: ## Push the .xpkg to the registry. GUARDED: requires ALLOW_PUBLISH=true.
	@[ "$(ALLOW_PUBLISH)" = "true" ] || { echo "Refusing to publish: provider is private/unverified. Re-run with ALLOW_PUBLISH=true VERSION=<vX.Y.Z> once approved."; exit 1; }
	@test -f $(XPKG_FILE) || { echo "$(XPKG_FILE) not found; run 'make xpkg VERSION=$(VERSION)' first"; exit 1; }
	$(CROSSPLANE) xpkg push --package-files=$(XPKG_FILE) $(XPKG_REF)
	@echo ">> pushed $(XPKG_REF)"
