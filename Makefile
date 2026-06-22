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
