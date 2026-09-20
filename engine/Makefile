BINARY  := devrites
DIST    := dist
GOFILES := $(shell find . -name '*.go' -not -path './$(DIST)/*')
# darwin/amd64 kept for Intel Macs; the five targets the release matrix ships.
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64

.PHONY: build fmt-check test test-race vet quality staticcheck govulncheck gosec lint cover crosscompile clean

STATICCHECK_VERSION := 2026.1
GOVULNCHECK_VERSION := v1.6.0
GOSEC_VERSION := v2.28.0
GOLANGCI_VERSION := v2.13.2
OSV_SCANNER_VERSION := v2.5.1
STATICCHECK := go run honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
GOVULNCHECK := go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
GOSEC := go run github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
GOLANGCI := go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION)
OSV_SCANNER := go run github.com/google/osv-scanner/v2/cmd/osv-scanner@$(OSV_SCANNER_VERSION)

build:
	CGO_ENABLED=0 go build -o $(BINARY) .

fmt-check:
	@test -z "$$(gofmt -l $(GOFILES))" || (echo "gofmt required:"; gofmt -l $(GOFILES); exit 1)

test:
	go test -shuffle=on ./...

# Race-enabled run for concurrent state paths and mutable test clients.
test-race:
	go test -race -shuffle=on -count=1 ./...

vet:
	go vet ./...

# quality is the full local gate. Mirror it in CI so a green `make quality`
# means a green pipeline (the date-sensitive parity golden aside; see ADR-0006).
quality: fmt-check vet staticcheck govulncheck gosec lint osv test-race

staticcheck:
	$(STATICCHECK) ./...

govulncheck:
	$(GOVULNCHECK) ./...

# No rule-class exclusions: G204/G304/G702/G703 sites carry per-call
# `// #nosec GXXX -- <justification>` annotations in the source, so a genuinely
# new injection or traversal sink fails this target.
gosec:
	$(GOSEC) -conf .gosec.json ./...

# Curated complement to staticcheck (errcheck + ineffassign; .golangci.yml).
lint:
	$(GOLANGCI) run ./...

# OSV.dev dependency scanner (mirrors CI dependency gate).
osv:
	$(OSV_SCANNER) scan --config ../osv-scanner.toml --lockfile go.mod

# Coverage baseline. No hard floor is wired yet (see ADR-0006): the number is
# understated because most engine behaviour is exercised through CLI-level
# integration tests in tests/, which don't attribute to per-package coverage.
cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | tail -1

# Prove the pure-Go binary cross-compiles to every shipped target with no CGo.
crosscompile:
	@mkdir -p $(DIST)
	@for p in $(PLATFORMS); do \
		os=$${p%/*}; arch=$${p#*/}; ext=""; \
		[ "$$os" = windows ] && ext=".exe"; \
		echo "building $$os/$$arch"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -o $(DIST)/$(BINARY)-$$os-$$arch$$ext . || exit 1; \
	done

clean:
	rm -rf $(BINARY) $(DIST)
