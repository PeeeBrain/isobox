# AGENTS.md

- Use project terms from [CONTEXT.md](CONTEXT.md).
- Keep changes small and reviewable.
- Run `gofmt -w .`, `go test ./...`, and `go build ./...` before handoff when code changes.
- Do not imply stronger isolation than the selected Runtime Backend provides.
- Do not create, move, delete, or push release tags without explicit user approval.
- Do not commit generated Task Records or `.isobox/tasks/`.
