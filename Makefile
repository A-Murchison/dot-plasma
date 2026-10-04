.PHONY: check test fmt vet

check: fmt vet test

fmt:
	gofmt -w ./cmd ./internal

vet:
	go vet ./...

test:
	go test ./...
