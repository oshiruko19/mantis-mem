BINARY  := mantis-mem
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
# Pinned linter version — `make lint` requires exactly this on PATH.
GOLANGCI_LINT_VERSION := 2.13.2
# Pure-Go SQLite (modernc) means no cgo: fully static, trivially cross-compiled.
export CGO_ENABLED := 0

# Release target matrix: os/arch pairs.
PLATFORMS := darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

.PHONY: build test e2e vet lint fmt tidy install clean release $(PLATFORMS)

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) .

test:
	go test ./...

# End-to-end tests (build tag e2e): compile the binary and drive it over stdio.
e2e:
	go test -tags e2e ./internal/mcpserver/...

vet:
	go vet ./...

# Requires golangci-lint v$(GOLANGCI_LINT_VERSION) exactly on PATH; fails with an
# install hint otherwise. Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)
lint:
	@command -v golangci-lint >/dev/null 2>&1 || { \
	  echo "golangci-lint not found — install the pinned version:"; \
	  echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)"; \
	  exit 1; }
	@have=$$(golangci-lint version 2>/dev/null | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1); \
	if [ "$$have" != "$(GOLANGCI_LINT_VERSION)" ]; then \
	  echo "golangci-lint version mismatch: found $$have, need $(GOLANGCI_LINT_VERSION)"; \
	  echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)"; \
	  exit 1; \
	fi
	golangci-lint run ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

install:
	go install -trimpath -ldflags "$(LDFLAGS)" .

clean:
	rm -rf $(BINARY) dist/

# Cross-compile every platform into dist/.
release: $(PLATFORMS)

$(PLATFORMS):
	@mkdir -p dist
	@os=$(word 1,$(subst /, ,$@)); arch=$(word 2,$(subst /, ,$@)); \
	ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
	out="dist/$(BINARY)-$(VERSION)-$$os-$$arch$$ext"; \
	echo "building $$out"; \
	GOOS=$$os GOARCH=$$arch go build -trimpath -ldflags "$(LDFLAGS)" -o "$$out" .
