.PHONY: fmt lint test build

# go-isobox-packages enumerates every Go package under the repository
# root that belongs to the isobox Go tree. It deliberately excludes
# vendored or fetched dependency trees, most notably
# `site/node_modules/`, which is the JS landing-page project and
# happens to ship a Go port of `flatted` as a transitive dependency.
# Including that tree in the isobox test/lint runs is misleading: the
# site is a separate project, those files are not isobox code, and
# they should not influence isobox's test or lint results.
go-isobox-packages:
	@go list ./... | grep -v '/node_modules/'

fmt:
	gofmt -w .

lint:
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)
	go vet ./...

test:
	go test $(shell make go-isobox-packages)

build:
	go build ./...
