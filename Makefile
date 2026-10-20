# govibe's commands. `make help` lists them.

SHELL := /bin/bash

# The template govibe embeds: its repository (a URL, or a local checkout) and
# the ref to take, a tag for a release of govibe.
TEMPLATE_REPO ?= https://github.com/ajlatour/golang-vibe-starter
REF           ?= main
SNAPSHOT      := internal/template/snapshot

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo devel)

.DEFAULT_GOAL := help
.PHONY: help sync build test e2e check lint

help: ## List the targets
	@awk 'BEGIN { FS = ":.*## " } /^[a-zA-Z_-]+:.*## / { printf "  make %-8s %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

sync: ## Take the template at REF (default main) from TEMPLATE_REPO into the snapshot govibe embeds
	rm -rf bin/template
	git clone -q $(TEMPLATE_REPO) bin/template
	git -C bin/template checkout -q $(REF)
	git -C bin/template archive --format=tar.gz HEAD > $(SNAPSHOT)/golang-vibe-starter.tar.gz
	@{ git -C bin/template describe --tags --exact-match 2>/dev/null || git -C bin/template rev-parse --short=12 HEAD; } > $(SNAPSHOT)/REF
	cd bin/template && go run ./cmd/initservice -list-features > $(CURDIR)/$(SNAPSHOT)/features.json
	@echo "The snapshot is golang-vibe-starter $$(cat $(SNAPSHOT)/REF)."

build: ## Build govibe into bin/govibe
	go build -ldflags "-X main.version=$(VERSION)" -o bin/govibe .

test: ## The unit tests
	go test -race ./...

e2e: build ## Make services with bin/govibe, as a user would, and test them (minutes)
	GOVIBE_E2E=1 go test -count=1 -timeout 30m -run TestEndToEnd ./...

check: ## gofmt, vet and the unit tests
	@unformatted="$$(gofmt -l .)"; if [ -n "$$unformatted" ]; then echo "gofmt: $$unformatted"; exit 1; fi
	go vet ./...
	$(MAKE) test

lint: ## staticcheck, every check
	go run honnef.co/go/tools/cmd/staticcheck@v0.8.2 -checks all ./...
