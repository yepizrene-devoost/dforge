# dforge Makefile

BINARY_NAME = dforge
BIN_DIR     = bin

# Channel marker only. A tag is injected when HEAD is exactly on it; every
# other revision keeps the `dev` marker, so the development tree never claims a
# released version. The commit identity is deliberately NOT injected here: the
# binary self-reports it from Go's own buildvcs stamp (vcs.revision,
# vcs.modified), keeping one source of truth per fact.
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || echo dev)

LDFLAGS = -s -w -X main.version=$(VERSION)

.PHONY: all
all: build

# Build the local binary with the version marker injected via ldflags.
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) (version: $(VERSION))..."
	go build -ldflags="$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY_NAME) .

# Cross-platform build (linux/darwin/windows x amd64/arm64, CGO disabled),
# for testing outside GoReleaser.
.PHONY: build-all
build-all:
	@mkdir -p "$(BIN_DIR)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-linux-amd64" .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-linux-arm64" .
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-darwin-amd64" .
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-darwin-arm64" .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-windows-amd64.exe" .
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="$(LDFLAGS)" -o "$(BIN_DIR)/$(BINARY_NAME)-windows-arm64.exe" .

.PHONY: test
test:
	go test ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: check
check: vet test

.PHONY: tidy
tidy:
	go mod tidy

# Install into $GOPATH/bin with the version marker injected.
.PHONY: install
install:
	go install -ldflags="$(LDFLAGS)" .

# Remove the local build output.
.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
