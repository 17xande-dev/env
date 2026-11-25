GOVULNCHECK_VERSION ?= v1.9.0

.PHONY: fmt-check vet test vulncheck

fmt-check:
	@files="$$(gofmt -l .)" && test -z "$$files" || { printf '%s\n' "$$files"; exit 1; }

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

vulncheck:
	GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	bin="$$(go env GOBIN)" && GOWORK=off "$${bin:-$$(go env GOPATH)/bin}/govulncheck" ./...
