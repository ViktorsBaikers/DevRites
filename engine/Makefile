BINARY  := devrites
DIST    := dist
GOFILES := $(shell find . -name '*.go' -not -path './$(DIST)/*')
# darwin/amd64 kept for Intel Macs; the five targets the release matrix ships.
PLATFORMS := darwin/arm64 darwin/amd64 linux/amd64 linux/arm64 windows/amd64

.PHONY: build fmt-check test test-race vet quality staticcheck govulncheck cover tools crosscompile clean

# Pinned analysis tools. `make tools` installs them into $(go env GOPATH)/bin;
# CI installs the same versions so local and CI verdicts match.
STATICCHECK_VERSION := 2025.1.1
GOVULNCHECK_VERSION := latest

build:
	CGO_ENABLED=0 go build -o $(BINARY) .

fmt-check:
	@test -z "$$(gofmt -l $(GOFILES))" || (echo "gofmt required:"; gofmt -l $(GOFILES); exit 1)

test:
	go test ./...

# Race-enabled run — the engine has concurrent index/source-cache paths; the
# detector is free signal on every test that touches them.
test-race:
	go test -race -count=1 ./...

vet:
	go vet ./...

# quality is the full local gate — mirror it in CI so a green `make quality`
# means a green pipeline (the date-sensitive parity golden aside; see ADR-0006).
quality: fmt-check vet staticcheck test-race

tools:
	go install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION)
	go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)

staticcheck:
	staticcheck ./...

govulncheck:
	govulncheck ./...

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
