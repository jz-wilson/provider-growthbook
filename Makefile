# ====================================================================================
# Setup Project
PROJECT_NAME := provider-growthbook
PROJECT_REPO := github.com/jz-wilson/crossplane-provider-growthbook

PLATFORMS ?= linux_amd64 linux_arm64
-include build/makelib/common.mk

# ====================================================================================
# Setup Output

-include build/makelib/output.mk

# ====================================================================================
# Setup Go

NPROCS ?= 1
GO_TEST_PARALLEL := $(shell echo $$(( $(NPROCS) / 2 )))
GO_STATIC_PACKAGES = $(GO_PROJECT)/cmd/provider
GO_LDFLAGS += -X $(GO_PROJECT)/internal/version.Version=$(VERSION)
GO_SUBDIRS += cmd internal apis
GO111MODULE = on
GOLANGCILINT_VERSION = 2.13.2
-include build/makelib/golang.mk

# ====================================================================================
# Setup Kubernetes tools

# NOTE(safe-start-fix): the crossplane-build submodule pins
# CROSSPLANE_CLI_VERSION to v1.20.0 (the legacy v1 "crank" binary), whose
# meta.pkg.crossplane.io/v1 Provider Go type predates the capabilities
# field. crank xpkg build unmarshals package/crossplane.yaml into that old
# type, silently dropping spec.capabilities, so the built xpkg never
# carries safe-start and Crossplane installs it without safe-start
# behavior. Pin to a v2 CLI release that knows about capabilities, and
# override the download recipe: the v2 CLI is published on
# cli.crossplane.io as "crossplane". Do not use releases.crossplane.io for
# it: that host serves the core controller binary at the same path, which
# has no xpkg subcommand.
CROSSPLANE_CLI_VERSION := v2.4.0

-include build/makelib/k8s_tools.mk

$(CROSSPLANE_CLI):
	@$(INFO) installing Crossplane CLI $(CROSSPLANE_CLI_VERSION)
	@curl -fsSLo $(CROSSPLANE_CLI) --create-dirs https://cli.crossplane.io/$(CROSSPLANE_CLI_CHANNEL)/$(CROSSPLANE_CLI_VERSION)/bin/$(SAFEHOST_PLATFORM)/crossplane?source=build || $(FAIL)
	@chmod +x $(CROSSPLANE_CLI)
	@$(OK) installing Crossplane CLI $(CROSSPLANE_CLI_VERSION)

# ====================================================================================
# Setup Images

IMAGES = provider-growthbook
-include build/makelib/imagelight.mk

# ====================================================================================
# Setup XPKG

XPKG_REG_ORGS ?= ghcr.io/jz-wilson
# NOTE(hasheddan): skip promoting on xpkg.upbound.io as channel tags are
# inferred.
XPKG_REG_ORGS_NO_PROMOTE ?= ghcr.io/jz-wilson
XPKGS = provider-growthbook
-include build/makelib/xpkg.mk

# NOTE(hasheddan): we force image building to happen prior to xpkg build so that
# we ensure image is present in daemon.
xpkg.build.provider-growthbook: do.build.images

fallthrough: submodules
	@echo Initial setup complete. Running make again . . .
	@make

# integration tests
e2e.run: test-integration

# Run integration tests.
test-integration: $(KIND) $(KUBECTL) $(CROSSPLANE_CLI) $(HELM3)
	@$(INFO) running integration tests using kind $(KIND_VERSION)
	@KIND_NODE_IMAGE_TAG=${KIND_NODE_IMAGE_TAG} $(ROOT_DIR)/cluster/local/integration_tests.sh || $(FAIL)
	@$(OK) integration tests passed

# Update the submodules, such as the common build scripts.
submodules:
	@git submodule sync
	@git submodule update --init --recursive

# NOTE(hasheddan): the build submodule currently overrides XDG_CACHE_HOME in
# order to force the Helm 3 to use the .work/helm directory. This causes Go on
# Linux machines to use that directory as the build cache as well. We should
# adjust this behavior in the build submodule because it is also causing Linux
# users to duplicate their build cache, but for now we just make it easier to
# identify its location in CI so that we cache between builds.
go.cachedir:
	@go env GOCACHE

go.mod.cachedir:
	@go env GOMODCACHE

# NOTE(hasheddan): we must ensure up is installed in tool cache prior to build
# as including the k8s_tools machinery prior to the xpkg machinery sets UP to
# point to tool cache.
build.init: $(CROSSPLANE_CLI)

# This is for running out-of-cluster locally, and is for convenience. Running
# this make target will print out the command which was used. For more control,
# try running the binary directly with different arguments.
run: go.build
	@$(INFO) Running Crossplane locally out-of-cluster . . .
	@# To see other arguments that can be provided, run the command with --help instead
	$(GO_OUT_DIR)/provider --debug

dev: $(KIND) $(KUBECTL)
	@$(INFO) Creating kind cluster
	@$(KIND) create cluster --name=$(PROJECT_NAME)-dev
	@$(KUBECTL) cluster-info --context kind-$(PROJECT_NAME)-dev
	@$(INFO) Installing Provider GrowthBook CRDs
	@$(KUBECTL) apply -R -f package/crds
	@$(INFO) Starting Provider GrowthBook controllers
	@$(GO) run cmd/provider/main.go --debug

dev-clean: $(KIND) $(KUBECTL)
	@$(INFO) Deleting kind cluster
	@$(KIND) delete cluster --name=$(PROJECT_NAME)-dev

.PHONY: submodules fallthrough test-integration run dev dev-clean

# ====================================================================================
# Special Targets

# Install gomplate
GOMPLATE_VERSION := 3.10.0
GOMPLATE := $(TOOLS_HOST_DIR)/gomplate-$(GOMPLATE_VERSION)

$(GOMPLATE):
	@$(INFO) installing gomplate $(SAFEHOSTPLATFORM)
	@mkdir -p $(TOOLS_HOST_DIR)
	@curl -fsSLo $(GOMPLATE) https://github.com/hairyhenderson/gomplate/releases/download/v$(GOMPLATE_VERSION)/gomplate_$(SAFEHOSTPLATFORM) || $(FAIL)
	@chmod +x $(GOMPLATE)
	@$(OK) installing gomplate $(SAFEHOSTPLATFORM)

export GOMPLATE

# This target prepares repo for your provider by replacing all "growthbook"
# occurrences with your provider name.
# This target can only be run once, if you want to rerun for some reason,
# consider stashing/resetting your git state.
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
provider.prepare:
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@PROVIDER=$(provider) ./hack/helpers/prepare.sh

# This target adds a new api type and its controller.
# You would still need to register new api in "apis/<provider>.go" and
# controller in "internal/controller/<provider>.go".
# Arguments:
#   provider: Camel case name of your provider, e.g. GitHub, PlanetScale
#   group: API group for the type you want to add.
#   kind: Kind of the type you want to add
#	apiversion: API version of the type you want to add. Optional and defaults to "v1alpha1"
provider.addtype: $(GOMPLATE)
	@[ "${provider}" ] || ( echo "argument \"provider\" is not set"; exit 1 )
	@[ "${group}" ] || ( echo "argument \"group\" is not set"; exit 1 )
	@[ "${kind}" ] || ( echo "argument \"kind\" is not set"; exit 1 )
	@PROVIDER=$(provider) GROUP=$(group) KIND=$(kind) APIVERSION=$(apiversion) PROJECT_REPO=$(PROJECT_REPO) ./hack/helpers/addtype.sh

define CROSSPLANE_MAKE_HELP
Crossplane Targets:
    submodules            Update the submodules, such as the common build scripts.
    run                   Run crossplane locally, out-of-cluster. Useful for development.

endef
# The reason CROSSPLANE_MAKE_HELP is used instead of CROSSPLANE_HELP is because the crossplane
# binary will try to use CROSSPLANE_HELP if it is set, and this is for something different.
export CROSSPLANE_MAKE_HELP

crossplane.help:
	@echo "$$CROSSPLANE_MAKE_HELP"

help-special: crossplane.help

.PHONY: crossplane.help help-special

# ====================================================================================
# Security

# govulncheck scans the module graph against the Go vulnerability database and
# reports only vulnerabilities reachable from this code. It runs as part of
# `make reviewable` so a known-vulnerable dependency fails the review gate.
GOVULNCHECK_VERSION ?= latest
go.vuln:
	@$(INFO) govulncheck
	@go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./... || $(FAIL)
	@$(OK) govulncheck

reviewable: go.vuln
.PHONY: go.vuln

# ====================================================================================
# Integration

# Runs the client integration tests against a live GrowthBook. Start one with
# `docker compose -f e2e/docker-compose.yml up -d --wait` and mint a key with
# `e2e/bootstrap.sh` first; the tests skip when GROWTHBOOK_API_KEY is unset.
test.integration:
	@$(INFO) go test integration
	@go test -tags integration -count=1 ./internal/clients/... || $(FAIL)
	@$(OK) go test integration
.PHONY: test.integration
