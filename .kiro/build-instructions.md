# Build Instructions

This project uses [Task](https://taskfile.dev) for build automation. Use Task commands instead of direct `go build` or `go test`.

## Commands

- `task build` — Build optimized binary with version metadata
- `task dev` — Fast development build (no optimization)
- `task test` — Run all tests with coverage
- `task lint` — Run linters (`go vet`)
- `task fmt` — Format code (`go fmt`)
- `task clean` — Remove build artifacts

## Notes

- Always use `task build` for production builds — it injects build date and commit hash via ldflags.
- `task dev` is faster for iteration but skips optimization flags.
- If Task is not available, fall back to: `go build ./cmd/kiro-krew`
