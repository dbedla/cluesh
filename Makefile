.PHONY: fmt vet test cover

fmt:
	gofmt -l -w ./src ./cmd

vet:
	go vet ./...

test:
	go test ./...

cover:
	go test ./src -coverprofile=cover.out
	go tool cover -func=cover.out | tail -1
