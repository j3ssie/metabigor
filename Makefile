BINARY    := metabigor
MODULE    := github.com/j3ssie/metabigor
VERSION   := $(shell cat internal/core/constants.go | grep 'VERSION =' | cut -d '"' -f 2)
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILDDATE := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS   := -s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.buildDate=$(BUILDDATE)'
GOFLAGS   := -trimpath
GOBIN_PATH := $(shell go env GOPATH)/bin

.PHONY: build test clean install build-all fmt vet lint e2e update release snapshot

build:
	@mkdir -p bin
	go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o bin/$(BINARY) ./cmd/metabigor
	@cp bin/$(BINARY) $(GOBIN_PATH)/$(BINARY)

install:
	go install $(GOFLAGS) -ldflags '$(LDFLAGS)' ./cmd/metabigor

test:
	go test -race -count=1 ./...

fmt:
	gofmt -w -s .

vet:
	go vet ./...

lint:
	@command -v golangci-lint >/dev/null 2>&1 || { echo "golangci-lint not installed. Install: https://golangci-lint.run/usage/install/"; exit 1; }
	golangci-lint run ./...

e2e: build
	@echo "Running end-to-end tests..."
	@cd test && ./run-e2e.sh

update:
	@echo "Updating embedded databases..."
	@wget -O public/ip-to-asn.csv.zip https://github.com/iplocate/ip-address-databases/raw/refs/heads/main/ip-to-asn/ip-to-asn.csv.zip
	@echo "ASN database updated at public/ip-to-asn.csv.zip"
	@wget -O public/ip-to-country.csv.zip https://github.com/iplocate/ip-address-databases/raw/refs/heads/main/ip-to-country/ip-to-country.csv.zip
	@echo "Country database updated at public/ip-to-country.csv.zip"
	@echo "All databases updated successfully"

clean:
	rm -rf bin/ dist/

build-all: clean
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-linux-amd64       ./cmd/metabigor
	GOOS=linux   GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-linux-arm64       ./cmd/metabigor
	GOOS=darwin  GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-darwin-amd64      ./cmd/metabigor
	GOOS=darwin  GOARCH=arm64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-darwin-arm64      ./cmd/metabigor
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o dist/$(BINARY)-windows-amd64.exe ./cmd/metabigor

snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Install: https://goreleaser.com/install/"; exit 1; }
	@echo "Building snapshot release with goreleaser..."
	goreleaser release --snapshot --clean --skip=publish

release:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Install: https://goreleaser.com/install/"; exit 1; }
	@echo "Creating release with goreleaser..."
	@echo "Note: Ensure you have a git tag and GITHUB_TOKEN set"
	goreleaser release --clean
