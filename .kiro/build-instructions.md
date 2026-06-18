# Build Instructions

This project uses [Task](https://taskfile.dev) for build automation.

## Rule

**Before running any build, test, format, or lint operation, check `Taskfile.yml` for a matching target and use it.** Do not invoke tools like `go build`, `go test`, `go fmt`, `go vet`, or `gofmt` directly when a Task target exists for that operation. Task targets encapsulate flags, paths, and configuration that direct invocations miss.

Run `task --list` to see all available targets.

## Examples

- Formatting code → `task fmt` (not `gofmt -w .`)
- Building → `task build` (not `go build ./cmd/...`)
- Running all tests → `task test` (not `go test ./...`)

## When direct commands are acceptable

- Task is not installed and cannot be installed
- You need flags the Task target doesn't support (e.g., running a single test by name with `-run`)
- The operation has no matching Task target
