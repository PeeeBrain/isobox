.PHONY: fmt lint test build

fmt:
	gofmt -w .

lint:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)
	go vet ./...

test:
	go test ./...

build:
	go build ./...
