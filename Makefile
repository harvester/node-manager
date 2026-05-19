ROOT              := $(realpath $(dir $(realpath $(firstword $(MAKEFILE_LIST)))))

MK_REPO_ID := $(shell echo -n "$(ROOT)$$(cat /etc/machine-id 2>/dev/null)" | sha256sum | cut -c1-8)
export MK_REPO_ID

MK_HOST_ARCH := $(shell uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

MK_DOCKER_PROGRESS ?= plain
export MK_DOCKER_PROGRESS

DOCKER_BUILDKIT := 1
export DOCKER_BUILDKIT

MK_DOCKER_RUN_OPTS_TTY := $(if $(CI),,-it)
export MK_DOCKER_RUN_OPTS_TTY

BUILD_FOR_CI ?=
export BUILD_FOR_CI

ifdef CI
  BOLD  :=
  CYAN  :=
  RESET :=
else
  BOLD  := \033[1m
  CYAN  := \033[36m
  RESET := \033[0m
endif

BANNER = @printf "$(BOLD)$(CYAN)[target: $@]$(RESET)\n"

DOCKER_BUILD = docker build \
    --progress=$(MK_DOCKER_PROGRESS) \
    --build-arg MK_REPO_ID \
    --build-arg MK_HOST_ARCH \
    -f $(ROOT)/Dockerfile $(ROOT)

.DEFAULT_GOAL := default

.PHONY: build test validate validate-ci generate generate-manifest package package-webhook \
        ci default release gen-version-env clean

# ---- gen-version-env ----
gen-version-env:
	@bash $(ROOT)/scripts/version > /dev/null

# ---- build ----
build: gen-version-env | $(ROOT)/bin
	$(BANNER)
	$(DOCKER_BUILD) --target build-output \
	    --output type=local,dest=$(ROOT)

# ---- test ----
test: gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target test

# ---- validate ----
validate: gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target validate

# ---- validate-ci ----
validate-ci: gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target validate-ci

# ---- generate ----
generate: gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target generate-output \
	    --output type=local,dest=$(ROOT)

# ---- generate-manifest ----
generate-manifest: gen-version-env
	$(BANNER)
	$(DOCKER_BUILD) --target generate-manifest-output \
	    --output type=local,dest=$(ROOT)

# ---- package ----
package: build
	$(BANNER)
	ARCH=$(MK_HOST_ARCH) $(ROOT)/scripts/package

# ---- package-webhook ----
package-webhook: build
	$(BANNER)
	ARCH=$(MK_HOST_ARCH) $(ROOT)/scripts/package-webhook

# ---- ci ----
ci: build test validate validate-ci

# ---- default ----
default: build test package package-webhook

# ---- release ----
release: ci

$(ROOT)/bin:
	mkdir -p $@

# ---- clean ----
clean:
	@rm -rf $(ROOT)/bin
	@rm -f $(ROOT)/scripts/.version_env
