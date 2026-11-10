# Copyright The OpenTelemetry Authors
# SPDX-License-Identifier: Apache-2.0

# Use bash for all shell commands (required for pipefail and other bash features)
SHELL := /bin/bash

.PHONY: all test test-unit test-integration test-e2e format lint build build/pkg build/instrumentation otelc clean \
        format/go format/yaml lint/go lint/go/fix lint/go/instrumentation lint/go/instrumentation/fix lint/yaml \
        lint/action lint/makefile lint/license-header lint/license-header/fix actionlint yamlfmt gotestfmt ratchet ratchet/pin \
        ratchet/update ratchet/check golangci-lint embedmd checkmake help docs \
        test-unit/race test-unit/pkg test-unit/instrumentation test-unit/helper \
        test-unit/coverage test-unit/pkg/coverage test-unit/instrumentation/coverage \
        check-coverage test-integration/coverage test-e2e/coverage test-latestlibbuild test-latestlibrun test-versionmatrix \
        weaver-install tidy/test-apps \
        fetch-upstream-semconv lint-schema \
        govulncheck govulncheck/instrumentation

# Constant variables
TOOLS_DIR = .tools
GO_VERSION = 1.26
INTEGRATION_TEST_RUN ?= .
PKG_COVERAGE_THRESHOLD ?= 70
RACE ?= 0
RACE_FLAG := $(if $(filter 1 true,$(RACE)),-race,)

# Modules/apps scanned by govulncheck for known Go CVEs (tool version pinned in
# .tools/go.mod, Renovate-managed like every other .tools binary).
# Core modules: root module plus every pkg/ module.
# Instrumentation is scanned separately via instrumented integration-test binaries
# (see govulncheck/instrumentation): source scans skip //go:build ignore files and
# cannot typecheck modules that need compile-time field injection (e.g. database/sql).
GOVULNCHECK_CORE_MODULES := . $(shell [ -d pkg ] && find pkg -type f -name 'go.mod' -exec dirname {} \; 2>/dev/null | sort)
# Top-level integration apps only (skip nested modules).
GOVULNCHECK_TEST_APPS := $(shell [ -d test/apps ] && find test/apps -mindepth 1 -maxdepth 1 -type d 2>/dev/null | sort)

# OTel Weaver execution for the local semantic-convention registry under
# schemas/otelc-contrib/. Weaver runs from an OCI image (no host install required);
# override OCI_BIN=podman or WEAVER_IMAGE=... to use a different runtime/version.
OCI_BIN ?= docker
WEAVER_IMAGE ?= otel/weaver:v0.19.0
OTELC_REGISTRY_DIR = $(CURDIR)/schemas/otelc-contrib

##@ Tooling

GOOS ?= $(shell go env GOOS)
EXT :=
ifeq ($(GOOS),windows)
	EXT = .exe
endif

TOOLS := $(CURDIR)/.bin

# Tools built from .tools module
$(TOOLS):
	@mkdir -p $@

$(TOOLS)/%: $(TOOLS_DIR)/go.mod | $(TOOLS)
	cd $(TOOLS_DIR) && \
	GOWORK=off go build -o $@ $(PACKAGE)

OTELC = $(TOOLS)/otelc$(EXT)
$(OTELC): PACKAGE=go.opentelemetry.io/otelc/tool/cmd/otelc

CROSSLINK = $(TOOLS)/crosslink$(EXT)
$(CROSSLINK): PACKAGE=go.opentelemetry.io/build-tools/crosslink

# Go tools built from .tools module (pinned versions in .tools/go.mod)
GOTESTFMT = $(TOOLS)/gotestfmt$(EXT)
$(GOTESTFMT): PACKAGE=github.com/gotesttools/gotestfmt/v2/cmd/gotestfmt

GOLANGCI_LINT = $(TOOLS)/golangci-lint$(EXT)
$(GOLANGCI_LINT): PACKAGE=github.com/golangci/golangci-lint/v2/cmd/golangci-lint

ACTIONLINT = $(TOOLS)/actionlint$(EXT)
$(ACTIONLINT): PACKAGE=github.com/rhysd/actionlint/cmd/actionlint

YAMLFMT = $(TOOLS)/yamlfmt$(EXT)
$(YAMLFMT): PACKAGE=github.com/google/yamlfmt/cmd/yamlfmt

RATCHET = $(TOOLS)/ratchet$(EXT)
$(RATCHET): PACKAGE=github.com/sethvargo/ratchet

EMBEDMD = $(TOOLS)/embedmd$(EXT)
$(EMBEDMD): PACKAGE=github.com/campoy/embedmd

CHECKMAKE = $(TOOLS)/checkmake$(EXT)
$(CHECKMAKE): PACKAGE=github.com/checkmake/checkmake/cmd/checkmake

GOVULNCHECK = $(TOOLS)/govulncheck$(EXT)
$(GOVULNCHECK): PACKAGE=golang.org/x/vuln/cmd/govulncheck

# Phony targets to build tools from .tools module (no go install; binaries in .bin/)
otelc: $(OTELC) ## Build otelc from .tools
gotestfmt: $(GOTESTFMT) ## Build gotestfmt from .tools
golangci-lint: $(GOLANGCI_LINT) ## Build golangci-lint from .tools
actionlint: $(ACTIONLINT) ## Build actionlint from .tools
yamlfmt: $(YAMLFMT) ## Build yamlfmt from .tools
ratchet: $(RATCHET) ## Build ratchet from .tools
embedmd: $(EMBEDMD) ## Build embedmd from .tools
checkmake: $(CHECKMAKE) ## Build checkmake from .tools

# Dynamic variables
ALL_GO_MOD_DIRS := $(shell find . -type f -name 'go.mod' -exec dirname {} \; | sort)

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show this help message
	@echo -e "\033[1;3;34mOpenTelemetry Go Compile Contrib.\033[0m\n"
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*##"; printf ""} /^[a-zA-Z_0-9\/-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

all: build format lint test

##@ Core Build

.ONESHELL:
build/pkg: ## Build all pkg modules to verify compilation
	@echo "Building pkg modules..."
	@set -euo pipefail
	@PKG_MODULES=$$(find pkg -name "go.mod" -type f -exec dirname {} \;); \
	for moddir in $$PKG_MODULES; do \
		echo "Building $$moddir..."; \
		(cd $$moddir && go mod tidy && go build ./...); \
	done
	@echo "All pkg modules built successfully"

.ONESHELL:
build/instrumentation: ## Build all instrumentation modules to verify compilation
	@echo "Building instrumentation modules..."
	@set -euo pipefail
	@INSTR_MODULES=$$(find instrumentation -name "go.mod" -type f -exec dirname {} \; | grep -v "instrumentation/runtime" | grep -v "instrumentation/database/sql"); \
	for moddir in $$INSTR_MODULES; do \
		echo "Building $$moddir..."; \
		(cd $$moddir && go mod tidy && go build ./...); \
	done
	@echo "All instrumentation modules built successfully"

build: build/pkg build/instrumentation ## Build all pkg and instrumentation modules

##@ Code Quality

format: ## Format Go code and YAML files
format: format/go format/yaml lint/license-header/fix

format/go: ## Format Go code only
format/go: $(GOLANGCI_LINT)
	@echo "Formatting Go code..."
	$(GOLANGCI_LINT) fmt --config .tools/golangci.yml

format/yaml: ## Format YAML files only (excludes testdata and schemas/otelc-contrib/.deps)
format/yaml: $(YAMLFMT)
	@echo "Formatting YAML files..."
	$(YAMLFMT) -conf .tools/yamlfmt -dstar \
		-exclude '**/schemas/otelc-contrib/.deps/**' \
		-exclude '**/testdata/**' \
		'**/*.yml' '**/*.yaml'

lint: ## Run all linters (Go, YAML, GitHub Actions, Makefile, typos)
lint: lint/go lint/yaml lint/action lint/makefile lint/license-header lint/typos

lint/action: ## Lint GitHub Actions workflows
lint/action: $(ACTIONLINT) ratchet/check
	@echo "Linting GitHub Actions workflows..."
	$(ACTIONLINT)

lint/go: ## Run golangci-lint on Go code
lint/go: $(GOLANGCI_LINT)
	@echo "Linting Go code..."
	$(GOLANGCI_LINT) run --config .tools/golangci.yml

lint/go/fix: ## Run golangci-lint on Go code and fix the issues
lint/go/fix: $(GOLANGCI_LINT)
	@echo "Linting Go code..."
	$(GOLANGCI_LINT) run --config .tools/golangci.yml --fix

lint/go/instrumentation: ## Lint new issues in instrumentation modules (golangci-lint only sees the root module via ./...)
lint/go/instrumentation: $(GOLANGCI_LINT)
	@echo "Linting instrumentation modules (new issues only)..."
	@set -euo pipefail
	@BASE_REV=$${LINT_BASE_REV:-origin/main}
	@CONFIG=$(CURDIR)/.tools/golangci.yml
	@INSTR_MODULES=$$(find instrumentation -name "go.mod" -type f -exec dirname {} \; | grep -v "instrumentation/runtime" | grep -v "instrumentation/database/sql"); \
	for moddir in $$INSTR_MODULES; do \
		echo "Linting $$moddir..."; \
		(cd $$moddir && $(GOLANGCI_LINT) run --config $$CONFIG --new-from-rev="$$BASE_REV" ./...); \
	done
	@echo "All instrumentation modules linted successfully"

lint/go/instrumentation/fix: ## Lint new issues in instrumentation modules and fix them
lint/go/instrumentation/fix: $(GOLANGCI_LINT)
	@echo "Fixing lint issues in instrumentation modules (new issues only)..."
	@set -euo pipefail
	@BASE_REV=$${LINT_BASE_REV:-origin/main}
	@CONFIG=$(CURDIR)/.tools/golangci.yml
	@INSTR_MODULES=$$(find instrumentation -name "go.mod" -type f -exec dirname {} \; | grep -v "instrumentation/runtime" | grep -v "instrumentation/database/sql"); \
	for moddir in $$INSTR_MODULES; do \
		echo "Fixing $$moddir..."; \
		(cd $$moddir && $(GOLANGCI_LINT) run --config $$CONFIG --new-from-rev="$$BASE_REV" --fix ./...); \
	done
	@echo "All instrumentation modules fixed successfully"

lint/yaml: ## Lint YAML formatting
lint/yaml: $(YAMLFMT)
	@echo "Linting YAML files..."
	$(YAMLFMT) -conf .tools/yamlfmt -lint -dstar \
		-exclude '**/schemas/otelc-contrib/.deps/**' \
		-exclude '**/testdata/**' \
		'**/*.yml' '**/*.yaml'

lint/makefile: ## Lint Makefile
lint/makefile: $(CHECKMAKE)
	@echo "Linting Makefile..."
	$(CHECKMAKE) --config .tools/checkmake Makefile

lint/license-header: ## Check license headers in source files
	@.github/scripts/license-check.sh

.PHONY: lint/license-header/fix
lint/license-header/fix: ## Add missing license headers to source files
	@.github/scripts/license-check.sh --fix

.PHONY: lint/typos
lint/typos: ## Check for typos using crate-ci/typos
	@echo "Checking for typos..."
	@if command -v typos >/dev/null 2>&1; then \
		typos --config .tools/typos.toml; \
	else \
		echo "Error: 'typos' not found on PATH."; \
		echo "Install with one of:"; \
		echo "  brew install typos-cli"; \
		echo "  cargo install typos-cli"; \
		echo "  https://github.com/crate-ci/typos/releases"; \
		echo "(The former ghcr.io/crate-ci/typos Docker image is no longer published.)"; \
		exit 1; \
	fi

##@ Markdown

.PHONY: lint/markdown
lint/markdown: ## Lint Check the markdown files.
	npx markdownlint-cli -c .tools/markdownlint.yaml **/*.md

.PHONY: lint/markdown/fix
lint/markdown/fix: ## Lint Check the markdown files and fix them.
	npx markdownlint-cli -c .tools/markdownlint.yaml --fix **/*.md

# Ratchet targets for GitHub Actions pinning

ratchet/pin: ## Pin GitHub Actions to commit SHAs
ratchet/pin: $(RATCHET)
	@echo "Pinning GitHub Actions to commit SHAs..."
	@find .github/workflows -name '*.yml' -o -name '*.yaml' | xargs $(RATCHET) pin

ratchet/update: ## Update pinned GitHub Actions to latest versions
ratchet/update: $(RATCHET)
	@echo "Updating pinned GitHub Actions to latest versions..."
	@find .github/workflows -name '*.yml' -o -name '*.yaml' | xargs $(RATCHET) update

ratchet/check: ## Verify all GitHub Actions are pinned
ratchet/check: $(RATCHET)
	@echo "Checking GitHub Actions are pinned..."
	@find .github/workflows -name '*.yml' -o -name '*.yaml' | xargs $(RATCHET) lint

##@ Documentation

docs: ## Update embedded documentation in markdown files
docs: $(EMBEDMD) tmp/make-help.txt
	@echo "Updating embedded documentation..."
	$(EMBEDMD) -w CONTRIBUTING.md README.md

tmp/make-help.txt: ## Generate make help output for embedding in documentation
tmp/make-help.txt: $(MAKEFILE_LIST)
	@mkdir -p tmp
	@$(MAKE) --no-print-directory help > tmp/make-help.txt

##@ Security

.ONESHELL:
govulncheck: $(GOVULNCHECK) ## Scan core modules (root, pkg) for known Go vulnerabilities
	@echo "Running govulncheck across $(words $(GOVULNCHECK_CORE_MODULES)) core modules..."
	@set -uo pipefail
	@status=0
	@for moddir in $(GOVULNCHECK_CORE_MODULES); do \
		echo "==> govulncheck $$moddir"; \
		(cd "$$moddir" && "$(GOVULNCHECK)" ./...) || status=1; \
	done; \
	if [ "$$status" -ne 0 ]; then echo "govulncheck: vulnerabilities found"; exit 1; fi; \
	echo "govulncheck: no reachable vulnerabilities found"

# Build each integration test app with otelc, then scan the resulting binary.
# Binary mode sees injected instrumentation (including //go:build ignore sources
# and field-injection modules like database/sql) that source mode cannot analyze.
.ONESHELL:
govulncheck/instrumentation: $(GOVULNCHECK) $(OTELC) build ## Scan instrumented test apps (binary mode) for known Go vulnerabilities
	@echo "Running govulncheck -mode=binary across $(words $(GOVULNCHECK_TEST_APPS)) instrumented test apps..."
	@set -uo pipefail
	@status=0
	@app_bin=app$(EXT)
	@otelc="$(OTELC)"
	@for appdir in $(GOVULNCHECK_TEST_APPS); do \
		app=$$(basename "$$appdir"); \
		echo "==> otelc go build $$app"; \
		if ! (cd "$$appdir" && "$$otelc" go build -a -o "$$app_bin" .); then \
			echo "govulncheck/instrumentation: failed to build $$app"; \
			status=1; \
			(cd "$$appdir" && "$$otelc" cleanup) || true; \
			continue; \
		fi; \
		echo "==> govulncheck -mode=binary $$app"; \
		if ! "$(GOVULNCHECK)" -mode=binary "$$appdir/$$app_bin"; then \
			status=1; \
		fi; \
		rm -f "$$appdir/$$app_bin"; \
		(cd "$$appdir" && "$$otelc" cleanup) || true; \
	done; \
	if [ "$$status" -ne 0 ]; then echo "govulncheck: vulnerabilities found or build failed"; exit 1; fi; \
	echo "govulncheck: no reachable vulnerabilities found"

##@ Testing

test: ## Run all tests (unit + integration + e2e)
test: test-unit test-integration test-e2e

test-unit: test-unit/pkg test-unit/instrumentation test-unit/helper ## Run all unit tests (pkg + instrumentation + test helpers)

test-unit/race: ## Run all unit tests with the Go race detector enabled
	$(MAKE) test-unit RACE=1

.ONESHELL:
test-unit/pkg: ## Run unit tests for pkg modules only
	@echo "Running pkg unit tests..."
	set -euo pipefail
	rm -f ./gotest-unit-pkg.log
	@PKG_MODULES=$$(find pkg -name "go.mod" -type f -exec dirname {} \;); \
	for moddir in $$PKG_MODULES; do \
    	if ! find "$$moddir" -name "*_test.go" -type f | grep -q .; then \
    		echo "Skipping $$moddir (no tests)..."; \
    		continue; \
    	fi; \
		echo "Testing $$moddir..."; \
		(cd "$$moddir" && go mod tidy); \
		go test -C "$$moddir" $(RACE_FLAG) -v -shuffle=on -timeout=5m -count=1 ./... 2>&1 | tee -a ./gotest-unit-pkg.log; \
	done

# Notes on test-unit/instrumentation implementation:
# - Excludes "runtime" entirely (compile-time field injection; no testable subpackages).
# - "database/sql" root package also needs field injection, so ./... cannot type-check
#   client.go. Safe subpackages (dsnparse, semconv) are tested explicitly afterwards.
# - Skips modules without test files to avoid empty test output.
# - Uses go test -C to run tests without changing directories (cleaner, more reliable).
# - Does NOT use gotestfmt because v2.5.0 has a bug that causes panics when go test
#   outputs build errors (JSON lines with ImportPath but no Package field).
#   Standard go test -v output is readable enough without formatting.
.ONESHELL:
test-unit/instrumentation: ## Run unit tests for instrumentation modules only
	@echo "Running instrumentation unit tests..."
	set -euo pipefail
	rm -f ./gotest-unit-instrumentation.log
	INSTR_MODULES=$$(find instrumentation -name "go.mod" -type f -exec dirname {} \; | grep -v "instrumentation/runtime" | grep -v "instrumentation/database/sql"); \
	for moddir in $$INSTR_MODULES; do \
		if ! find "$$moddir" -name "*_test.go" -type f | grep -q .; then \
			echo "Skipping $$moddir (no tests)..."; \
			continue; \
		fi; \
		echo "Testing $$moddir..."; \
		(cd "$$moddir" && go mod tidy); \
		go test -C "$$moddir" $(RACE_FLAG) -v -shuffle=on -timeout=5m -count=1 ./... 2>&1 | tee -a ./gotest-unit-instrumentation.log; \
	done
	# database/sql: root package references otelc-injected fields on database/sql
	# types and cannot be built uninstrumented. dsnparse/semconv do not.
	echo "Testing instrumentation/database/sql/otelcsql (dsnparse, semconv)..."
	(cd instrumentation/database/sql/otelcsql && go mod tidy)
	go test -C instrumentation/database/sql/otelcsql $(RACE_FLAG) -v -shuffle=on -timeout=5m -count=1 ./dsnparse/... ./semconv/... 2>&1 | tee -a ./gotest-unit-instrumentation.log

.ONESHELL:
test-unit/helper: ## Run unit tests for test helper packages
	@echo "Running test helper unit tests..."
	set -euo pipefail
	rm -f ./gotest-unit-helper.log
	go test -C "test" $(RACE_FLAG) -v -shuffle=on -timeout=5m -count=1 ./testutil/... 2>&1 | tee ./gotest-unit-helper.log

test-unit/coverage: test-unit/pkg/coverage test-unit/instrumentation/coverage ## Run all unit tests with coverage

.ONESHELL:
test-unit/pkg/coverage: ## Run unit tests with coverage for pkg modules only
	@echo "Running pkg unit tests with coverage..."
	set -euo pipefail
	rm -f ./gotest-unit-pkg.log
	@PKG_MODULES=$$(find pkg -name "go.mod" -type f -exec dirname {} \;); \
	for moddir in $$PKG_MODULES; do \
		if ! find "$$moddir" -name "*_test.go" -type f | grep -q .; then \
			echo "Skipping $$moddir (no tests)..."; \
			continue; \
		fi; \
		echo "Testing $$moddir with coverage..."; \
		(cd "$$moddir" && go mod tidy); \
		go test -C "$$moddir" -v -shuffle=on -timeout=5m -count=1 ./... -coverprofile=coverage.txt -covermode=atomic 2>&1 | tee -a ./gotest-unit-pkg.log; \
	done
	@echo "Merging coverage files into coverage-pkg.txt..."
	@echo "mode: atomic" > coverage-pkg.txt
	@find pkg -name "coverage.txt" -exec grep -h -v "^mode:" {} \; >> coverage-pkg.txt 2>/dev/null || true
	@find pkg -name "coverage.txt" -delete 2>/dev/null || true

# Same implementation as test-unit/instrumentation but with coverage flags.
# Coverage files from each module are merged into a single coverage-instrumentation.txt file.
.ONESHELL:
test-unit/instrumentation/coverage: ## Run unit tests with coverage for instrumentation modules only
	@echo "Running instrumentation unit tests with coverage..."
	set -euo pipefail
	rm -f ./gotest-unit-instrumentation.log
	INSTR_MODULES=$$(find instrumentation -name "go.mod" -type f -exec dirname {} \; | grep -v "instrumentation/runtime" | grep -v "instrumentation/database/sql"); \
	for moddir in $$INSTR_MODULES; do \
		if ! find "$$moddir" -name "*_test.go" -type f | grep -q .; then \
			echo "Skipping $$moddir (no tests)..."; \
			continue; \
		fi; \
		echo "Testing $$moddir with coverage..."; \
		(cd "$$moddir" && go mod tidy); \
		go test -C "$$moddir" -v -shuffle=on -timeout=5m -count=1 ./... -coverprofile=coverage.txt -covermode=atomic 2>&1 | tee -a ./gotest-unit-instrumentation.log; \
	done
	# See test-unit/instrumentation: only packages that type-check without field injection.
	echo "Testing instrumentation/database/sql/otelcsql (dsnparse, semconv) with coverage..."
	(cd instrumentation/database/sql/otelcsql && go mod tidy)
	go test -C instrumentation/database/sql/otelcsql -v -shuffle=on -timeout=5m -count=1 ./dsnparse/... ./semconv/... -coverprofile=coverage.txt -covermode=atomic 2>&1 | tee -a ./gotest-unit-instrumentation.log
	@echo "Merging coverage files into coverage-instrumentation.txt..."
	@echo "mode: atomic" > coverage-instrumentation.txt
	@find instrumentation -name "coverage.txt" -exec grep -h -v "^mode:" {} \; >> coverage-instrumentation.txt 2>/dev/null || true
	@find instrumentation -name "coverage.txt" -delete 2>/dev/null || true

.ONESHELL:
check-coverage: test-unit/pkg/coverage ## Verify the unit test coverage floor for pkg modules
	@echo "Checking unit test coverage floors..."
	set -euo pipefail
	coverage_file="coverage-pkg.txt"; \
	threshold="$(PKG_COVERAGE_THRESHOLD)"; \
	if [[ ! -f "$$coverage_file" ]]; then \
		echo "Missing coverage report: $$coverage_file"; \
		exit 1; \
	fi; \
	coverage_value=$$(awk '/^mode:/ { next } { total += $$2; if ($$3 > 0) covered += $$2 } END { if (total > 0) printf "%.1f", (covered / total) * 100; else print "0.0" }' "$$coverage_file"); \
	if [[ -z "$$coverage_value" ]]; then \
		echo "Unable to determine coverage for $$coverage_file"; \
		exit 1; \
	fi; \
	awk -v coverage="$$coverage_value" -v threshold="$$threshold" 'BEGIN { exit (coverage + 0 >= threshold + 0) ? 0 : 1 }' || { \
		echo "Coverage $$coverage_file is below $$threshold%: $$coverage_value%"; \
		exit 1; \
	}; \
	echo "$$coverage_file: $$coverage_value% (threshold $$threshold%)"

.ONESHELL:
test-integration: go-protobuf-plugins ## Run integration tests
test-integration: build $(OTELC)
	@echo "Running integration tests..."
	set -euo pipefail
	# 40m: linodego public-method instrumentation rewrites ~450 *Client methods per
	# instrumented build; under coverage (all tests, no shards) wall time exceeds 20m.
	go -C "test" test -json -v -shuffle=on -timeout=40m -count=1 -tags integration -run '$(value INTEGRATION_TEST_RUN)' ./integration/... 2>&1 | tee ./gotest-integration.log

.ONESHELL:
test-latestlibbuild: build $(OTELC) ## Run LatestLibBuild tests
	@echo "Running LatestLibBuild tests..."
	set -euo pipefail
	go -C "test" test -json -v -shuffle=on -timeout=20m -count=1 -tags latestlibbuild ./latestlibbuild/... 2>&1 | tee ./gotest-latestlibbuild.log

.ONESHELL:
test-latestlibrun: build $(OTELC) ## Run LatestLibRun tests (bump apps to @latest then run integration suite)
	@echo "Bumping test apps to @latest..."
	set -euo pipefail
	go -C "test" test -json -v -shuffle=on -timeout=10m -count=1 -tags latestlibrun ./latestlibrun/... 2>&1 | tee ./gotest-latestlibrun.log
	$(MAKE) tidy/test-apps
	@echo "Syncing test module with bumped apps..."
	go -C "test" mod tidy
	@echo "Running integration suite against @latest deps..."
	$(MAKE) test-integration

.ONESHELL:
test-versionmatrix: build $(OTELC) ## Run VersionMatrix tests (pin apps to each rule's bounds then run integration suite, once per tier)
	@echo "Computing version matrix tiers..."
	set -euo pipefail
	rm -f ./gotest-versionmatrix.log
	tiers=$$(go -C "test" test -v -count=1 -tags versionmatrix -run '^TestVersionMatrixTierCount$$' ./versionmatrix/... 2>&1 | grep -oE 'VERSIONMATRIX_TIERS=[0-9]+' | tail -1 | cut -d= -f2)
	echo "Version matrix needs $$tiers tier(s)"
	for (( tier=0; tier<tiers; tier++ )); do
		echo "Pinning test apps to per-rule bound tier $$tier..."
		VERSIONMATRIX_TIER=$$tier go -C "test" test -json -v -shuffle=on -timeout=10m -count=1 -tags versionmatrix -run '^TestVersionMatrixBump$$' ./versionmatrix/... 2>&1 | tee -a ./gotest-versionmatrix.log
		$(MAKE) tidy/test-apps
		echo "Syncing test module with pinned apps..."
		go -C "test" mod tidy
		echo "Running integration suite against bound tier $$tier..."
		$(MAKE) test-integration
	done

.ONESHELL:
test-integration/coverage: go-protobuf-plugins ## Run integration tests with coverage report
test-integration/coverage: build $(OTELC)
	@echo "Running integration tests with coverage report..."
	set -euo pipefail
	# See test-integration: linodego builds need >20m when the suite is unsharded.
	go -C "test" test -json -v -shuffle=on -timeout=40m -count=1 -tags integration ./integration/... -coverprofile=../coverage-integration.txt -covermode=atomic 2>&1 | tee ./gotest-integration.log

.ONESHELL:
test-e2e: ## Run e2e tests
test-e2e: build $(OTELC)
	@echo "Running e2e tests..."
	set -euo pipefail
	go -C "test" test -json -v -shuffle=on -timeout=30m -count=1 -tags e2e ./e2e/... 2>&1 | tee ./gotest-e2e.log

.ONESHELL:
test-e2e/coverage: ## Run e2e tests with coverage report
test-e2e/coverage: build $(OTELC)
	@echo "Running e2e tests with coverage report..."
	set -euo pipefail
	go -C "test" test -json -v -shuffle=on -timeout=30m -count=1 -tags e2e ./e2e/... -coverprofile=../coverage-e2e.txt -covermode=atomic 2>&1 | tee ./gotest-e2e.log

.PHONY: crosslink
crosslink: $(CROSSLINK) ## Update intra-repository dependencies in all go modules
	@# Clean .otelc-build directories before generating go.work to avoid parsing generated go.mod
	@find . -type d -name ".otel*-build*" -exec rm -rf {} + 2>/dev/null || true
	@find . -type f -name ".otel*-build*.lock" -delete 2>/dev/null || true
	@echo "Updating intra-repository dependencies in all go modules" \
		&& $(CROSSLINK) --root=$(CURDIR)

.PHONY: go-work
go-work: $(CROSSLINK) ## Generate go.work file for local development
	@echo "Generating go.work file for local development..."
	@$(CROSSLINK) work --root=$(CURDIR) --go=$(GO_VERSION)
	@# Fix go version to include patch version (crosslink only supports major.minor)
	@sed -i.bak 's/^go $(GO_VERSION)$$/go $(GO_VERSION).0/' go.work && rm -f go.work.bak
	@# Drop tool-only modules: their transitive deps conflict with the main modules
	@# (e.g. old monolithic genproto vs. split genproto/googleapis/rpc).
	@go work edit -dropuse ./.tools
	@go work edit -dropuse ./.github/tools
	@echo "go.work file generated successfully"

.PHONY: go-mod-tidy
go-mod-tidy: $(ALL_GO_MOD_DIRS:%=go-mod-tidy/%) ## Run go mod tidy in all modules

go-mod-tidy/%: DIR=$*
go-mod-tidy/%: crosslink
	@echo "Running go mod tidy in $(DIR)" \
		&& cd $(DIR) \
		&& go mod tidy

##@ Utilities

clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf dist .bin
	find . -type d -name ".otel*-build*" -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name ".otel*-build*.lock" -delete 2>/dev/null || true
	find . -type f \( -name gotest-unit-pkg.log -o -name gotest-unit-instrumentation.log -o -name gotest-unit-helper.log -o -name gotest-integration.log -o -name gotest-e2e.log -o -name gotest-latestlibbuild.log -o -name gotest-latestlibrun.log -o -name gotest-versionmatrix.log -o -name coverage-pkg.txt -o -name coverage-instrumentation.txt -o -name coverage-integration.txt -o -name coverage-e2e.txt \) -delete

.ONESHELL:
tidy/test-apps: ## Run go mod tidy in all test app modules
	@echo "Running go mod tidy in test app modules..."
	@set -euo pipefail
	@TEST_APP_MODULES=$$(find test/apps -name "go.mod" -type f -exec dirname {} \;); \
	for moddir in $$TEST_APP_MODULES; do \
		echo "Tidying $$moddir..."; \
		(cd "$$moddir" && go mod tidy); \
	done
	@echo "All test app modules tidied successfully"

go-protobuf-plugins: ## Install Go protobuf plugins if not present
	@if ! command -v protoc-gen-go >/dev/null 2>&1; then \
		echo "Installing Go protobuf plugins..."; \
		go install google.golang.org/protobuf/cmd/protoc-gen-go@latest; \
	fi
	@if ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then \
		echo "Installing Go protobuf gRPC plugins..."; \
		go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest; \
	fi

# Semantic Convention Registry targets

weaver-install: ## Install OTel Weaver if not present
	@if ! command -v weaver >/dev/null 2>&1; then \
		echo "Installing OTel Weaver..."; \
		WEAVER_VERSION="v0.19.0"; \
		if [ "$$(uname -s)" = "Darwin" ]; then \
			if [ "$$(uname -m)" = "arm64" ]; then \
				WEAVER_ARCH="aarch64-apple-darwin"; \
			else \
				WEAVER_ARCH="x86_64-apple-darwin"; \
			fi; \
		elif [ "$$(uname -s)" = "Linux" ]; then \
			WEAVER_ARCH="x86_64-unknown-linux-gnu"; \
		else \
			echo "Error: Unsupported platform $$(uname -s)"; \
			exit 1; \
		fi; \
		WEAVER_URL="https://github.com/open-telemetry/weaver/releases/download/$${WEAVER_VERSION}/weaver-$${WEAVER_ARCH}.tar.xz"; \
		echo "Downloading weaver from $${WEAVER_URL}..."; \
		mkdir -p /tmp/weaver-install; \
		curl -fsSL "$${WEAVER_URL}" -o /tmp/weaver-install/weaver.tar.xz; \
		tar -xJf /tmp/weaver-install/weaver.tar.xz -C /tmp/weaver-install; \
		WEAVER_BIN=$$(find /tmp/weaver-install -name weaver -type f); \
		if [ -z "$$WEAVER_BIN" ]; then \
			echo "Error: weaver binary not found in archive"; \
			rm -rf /tmp/weaver-install; \
			exit 1; \
		fi; \
		chmod +x "$$WEAVER_BIN"; \
		mkdir -p "$$(go env GOPATH)/bin"; \
		mv "$$WEAVER_BIN" "$$(go env GOPATH)/bin/weaver"; \
		rm -rf /tmp/weaver-install; \
		echo "Installed weaver to $$(go env GOPATH)/bin/weaver"; \
		weaver --version; \
	else \
		echo "OTel Weaver is already installed at $$(command -v weaver)"; \
		weaver --version; \
	fi

# Semantic Conventions Validation Targets
#
# The project's telemetry contract lives in the local Weaver registry under
# schemas/otelc-contrib/ (see docs/semantic-conventions.md). Weaver runs from an OCI
# image ($(WEAVER_IMAGE)) via $(OCI_BIN), so no host install is required for
# these targets.

fetch-upstream-semconv: ## Pre-fetch the pinned upstream semconv registry into schemas/otelc-contrib/.deps/
	@scripts/semconv/fetch-upstream-semconv.sh

lint-schema: ## Validate the local semantic-convention registry (schemas/otelc-contrib/) with OTel Weaver
lint-schema: fetch-upstream-semconv
	@echo "Validating otelc semantic-convention registry (schemas/otelc-contrib)..."
	@# Guard: the upstream dependency pinned in the registry manifest must match .semconv-version.
	@MANIFEST_VERSION=$$(grep -oE 'upstream-v[0-9]+\.[0-9]+\.[0-9]+' "$(OTELC_REGISTRY_DIR)/registry_manifest.yaml" | head -1 | sed -E 's/upstream-v//'); \
	SEMCONV_VERSION=$$(grep -E '^v[0-9]+\.[0-9]+\.[0-9]+' .semconv-version | head -1 | tr -d '[:space:]' | sed 's/^v//'); \
	if [ -z "$$MANIFEST_VERSION" ]; then \
		echo "::error::Could not read the upstream version from $(OTELC_REGISTRY_DIR)/registry_manifest.yaml"; \
		exit 1; \
	fi; \
	if [ "$$MANIFEST_VERSION" != "$$SEMCONV_VERSION" ]; then \
		echo "::error::registry_manifest.yaml pins upstream v$$MANIFEST_VERSION but .semconv-version is v$$SEMCONV_VERSION - bump them together"; \
		exit 1; \
	fi; \
	echo "Upstream semconv dependency: v$$MANIFEST_VERSION (matches .semconv-version)"
	@scripts/semconv/lint-schema.sh $(OCI_BIN) $(WEAVER_IMAGE) "$(OTELC_REGISTRY_DIR)"

# `lint/semantic-conventions` is the umbrella entry point used by CI and the
# top-level `lint` target; it now validates the project's own registry.
lint/semantic-conventions: ## Validate the otelc semantic-convention registry (schemas/otelc-contrib/) with OTel Weaver
lint/semantic-conventions: lint-schema

semantic-conventions/diff: ## Generate diff between current version and latest (non-blocking informational check)
semantic-conventions/diff: weaver-install
	@echo "Generating semantic convention registry diff (current vs latest)..."
	@mkdir -p tmp
	@# Read the semconv version from .semconv-version file (ignore comments and empty lines)
	@if [ ! -f .semconv-version ]; then \
		echo "Error: .semconv-version file not found"; \
		exit 1; \
	fi; \
	CURRENT_VERSION=$$(grep -E '^v[0-9]+\.[0-9]+\.[0-9]+' .semconv-version | head -1 | tr -d '[:space:]'); \
	if [ -z "$$CURRENT_VERSION" ]; then \
		echo "Error: No version found in .semconv-version file"; \
		exit 1; \
	fi; \
	echo "Current project version: $$CURRENT_VERSION"; \
	echo "Cloning semantic-conventions repositories..."; \
	rm -rf /tmp/semconv-current-$$$$ /tmp/semconv-latest-$$$$ tmp/registry-diff-latest; \
	git clone --depth 1 --branch $$CURRENT_VERSION https://github.com/open-telemetry/semantic-conventions.git /tmp/semconv-current-$$$$ 2>/dev/null && \
	git clone --depth 1 https://github.com/open-telemetry/semantic-conventions.git /tmp/semconv-latest-$$$$ 2>/dev/null || { \
		echo "⚠️  Warning: Failed to clone repositories (this is non-blocking)"; \
		echo "⚠️  Registry diff generation failed." > tmp/registry-diff-latest.md; \
		rm -rf /tmp/semconv-current-$$$$ /tmp/semconv-latest-$$$$; \
		exit 0; \
	}; \
	mkdir -p tmp/registry-diff-latest; \
	weaver registry diff \
		--registry /tmp/semconv-latest-$$$$/model \
		--baseline-registry /tmp/semconv-current-$$$$/model \
		--diff-format markdown \
		--output tmp/registry-diff-latest || { \
			echo "⚠️  Warning: Registry diff generation failed (this is non-blocking)"; \
			rm -rf tmp/registry-diff-latest; \
			echo "⚠️  Registry diff generation failed." > tmp/registry-diff-latest.md; \
		}; \
	rm -rf /tmp/semconv-current-$$$$ /tmp/semconv-latest-$$$$; \
	if [ -f tmp/registry-diff-latest/diff.md ]; then \
		mv tmp/registry-diff-latest/diff.md tmp/registry-diff-latest.md; \
		rm -rf tmp/registry-diff-latest; \
		echo ""; \
		echo "🆕 Available updates (latest vs $$CURRENT_VERSION):"; \
		echo "Saved to: tmp/registry-diff-latest.md"; \
		echo ""; \
		cat tmp/registry-diff-latest.md; \
	elif [ -f tmp/registry-diff-latest.md ]; then \
		echo ""; \
		echo "⚠️  Registry diff generation failed."; \
		cat tmp/registry-diff-latest.md; \
	fi; \
	exit 0

semantic-conventions/resolve: ## Display the current semantic conventions version
semantic-conventions/resolve:
	@echo "Semantic conventions version management"
	@echo "========================================"
	@if [ ! -f .semconv-version ]; then \
		echo "Error: .semconv-version file not found"; \
		exit 1; \
	fi; \
	CURRENT_VERSION=$$(grep -E '^v[0-9]+\.[0-9]+\.[0-9]+' .semconv-version | head -1 | tr -d '[:space:]'); \
	if [ -z "$$CURRENT_VERSION" ]; then \
		echo "Error: No version found in .semconv-version file"; \
		exit 1; \
	fi; \
	echo "Current version: $$CURRENT_VERSION"; \
	echo ""; \
	echo "Checking for latest version..."; \
	LATEST_TAG=$$(git ls-remote --tags --refs https://github.com/open-telemetry/semantic-conventions.git 2>/dev/null | \
		grep -E 'refs/tags/v[0-9]+\.[0-9]+\.[0-9]+$$' | \
		awk -F/ '{print $$NF}' | \
		sort -t. -k1,1n -k2,2n -k3,3n | \
		tail -1); \
	if [ -n "$$LATEST_TAG" ]; then \
		echo "Latest available: $$LATEST_TAG"; \
		if [ "$$CURRENT_VERSION" != "$$LATEST_TAG" ]; then \
			echo ""; \
			echo "🆕 Update available: $$CURRENT_VERSION → $$LATEST_TAG"; \
		else \
			echo "✅ You are using the latest version"; \
		fi; \
	else \
		echo "⚠️  Unable to check latest version"; \
	fi
