# Contributing

This guide is intentionally short. Prefer the existing project language and
small, reviewable changes.

## Local workflow

Run these before handing off code changes:

```sh
gofmt -w .
go test ./...
go build ./...
```

For release-sensitive changes, also verify Linux builds:

```sh
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/isobox-linux-amd64 ./cmd/isobox
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o /tmp/isobox-linux-arm64 ./cmd/isobox
```

## Project language

Use the domain terms in [CONTEXT.md](CONTEXT.md). In particular:

- Use **Task**, **Task Attempt**, **Task Record**, **Task Result**, **Sandbox**,
  **Runtime Backend**, **Workspace**, and **Promotion** consistently.
- Avoid implying stronger isolation than the selected Runtime Backend provides.
- Treat Promotion and release tagging as explicit human-approved actions.

## Pull requests

- Keep changes scoped to one behavior or doc improvement.
- Include tests for behavior changes.
- Update `README.md`, `CHANGELOG.md`, or `docs/` when user-visible behavior,
  commands, installation, or release mechanics change.
- Do not commit generated Task Records or `.isobox/tasks/`.

## Releases

Follow [docs/releasing.md](docs/releasing.md).

Creating, moving, deleting, or pushing release tags is a human-in-the-loop
process. Agents **MUST NOT** perform those actions unless the user explicitly
requests a release or approves an agent-recommended release first.
