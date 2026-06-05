BINARY ?= abp-cli
DIST_DIR ?= dist
RANGE ?= HEAD
MODULE := $(shell go list -m)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo local)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS ?= -s -w -X $(MODULE)/internal/abpcrud.Version=$(VERSION) -X $(MODULE)/internal/abpcrud.Commit=$(COMMIT) -X $(MODULE)/internal/abpcrud.BuildDate=$(BUILD_DATE)

.PHONY: test build install templates changelog release-check clean

test:
	go test ./...

build:
	mkdir -p $(DIST_DIR)
	go build -trimpath -ldflags="$(LDFLAGS)" -o $(DIST_DIR)/$(BINARY) .

install:
	go install .

templates:
	go run . templates --out ./abp-templates

changelog:
	sh ./scripts/update-changelog.sh $(VERSION) "$(RANGE)"

release-check:
	sh ./scripts/check-release.sh

clean:
	rm -rf $(DIST_DIR)
