# Development Guidelines for fs-tracer

## Workflow
- Follow TDD: write failing tests first (Red), implement minimally (Green), then refactor.
- Prefer pure functions for:
  - fs_usage line parsing
  - aggregation/classification/filters
  - output generation (text/JSON/sandbox snippet)
  - argument handling (cobra-driven; keep data in `args.Options`)
- Keep I/O and concurrency in `main` (or `app.Run`) only.

## Testing
### Unit tests
- Filters (allow/ignore/prefix, max-depth, dirs-only)
+- fs_usage line parser
  aggregation/classification
  output formatting (events, split-access, JSON)
  sandbox snippet generation

### Integration tests (with fake fs_usage)
- Use `FsUsageRunner` fake that streams sample logs.
- Cover major modes: `--events --json`, `--split-access`, `--sandbox-snippet`, `--raw`, `--no-pid-filter`, `--max-depth`.

### CI
- `go test ./...`

## Notes
- Error codes: propagate yourcmd exit code; internal errors 90–99.
- fs_usage invocation: `sudo fs_usage -w -f filesys,pathname <pid>` (or without sudo when `--no-sudo`).
- Keep options centralized in cobra; avoid duplicate flag definitions elsewhere.
- Thread IDs via Mach are not available in SIP/macOS 15+ environments; follow-children filtering should rely on descendant PIDs and comm names. When thread IDs are unavailable, encourage `--allow-process` to reduce noise.
