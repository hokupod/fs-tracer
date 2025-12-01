# fs-tracer

macOS-only file access tracer that runs your command as the original user and parses `fs_usage` output to list touched paths. Designed to gather material for sandbox-exec profiles.

## Install
```sh
go install github.com/hokupod/fs-tracer/cmd/fs-tracer@latest
```

## Quick start
```sh
fs-tracer -- yourcmd [args...]
```

Examples:
```sh
fs-tracer -- ls -la /usr
fs-tracer -v -- /usr/local/bin/mytool --config=config.yml
fs-tracer --json --split-access -- /usr/bin/curl https://example.com
```

## Options
- `-v, --events`          : emit event log (time/pid/comm/op/path), no sorting
- `--json`                : JSON output (events -> 1 JSON per line, default -> array)
- `--split-access`        : separate read/write sets
- `--sandbox-snippet`     : emit sandbox-exec s-expressions (mutually exclusive with `--events`)
- `--dirs`, `--prefix-only`: output parent directories instead of full paths
- `--ignore-process NAME` : drop events from process name (repeatable)
- `--ignore-prefix PATH`  : drop events whose path starts with prefix (repeatable)
- `--no-sudo`             : run `fs_usage` without sudo (fs-tracer must then be root, yourcmd still runs as original UID/GID)
- `--raw`                 : disable ignore-process/prefix filters
- `--no-pid-filter`       : do **not** restrict to yourcmd’s PID (useful for multi-process tools; noise will increase)
- `--ignore-cwd`          : ignore events under the current working directory (also expands `.` in ignore-prefix to cwd)
- `--max-depth N`         : truncate paths to at most N components (0 = unlimited, aggregation happens before output/sandbox)

Env for debugging:
- `FS_TRACER_DEBUG=1`     : print raw `fs_usage:` lines and parse errors to stderr
- `FS_TRACER_DEBUG_ALL=1` : disable PID filter (same effect as `--no-pid-filter`) in debug runs

Exit codes: yourcmd’s exit code is propagated; internal errors use 90–99.

## Shell completion
Use the built-in cobra completion command:
```sh
fs-tracer completion bash > /etc/bash_completion.d/fs-tracer
fs-tracer completion zsh  > /usr/local/share/zsh/site-functions/_fs-tracer
fs-tracer completion fish > ~/.config/fish/completions/fs-tracer.fish
fs-tracer completion powershell > fs-tracer.ps1  # then import in your profile
```

## How it works (and why PID filter exists)
`fs_usage` is started **with** the target PID (`fs_usage -w -f filesys,pathname <pid>`), so kernel-side tracing is already narrowed to your command. fs-tracer then applies an in-process PID filter (default ON) and the other filters (allow/ignore/process/prefix, max-depth). `--no-pid-filter` only disables the Go-side PID check; it does **not** widen fs_usage’s kernel tracing to other PIDs. Use `--allow-process`/`--ignore-*`/`--max-depth` to further shape what gets emitted.

## Known limitations (fs_usage / macOS)
- **SIP-protected platform binaries** (Apple-provided commands) sometimes emit no events to dtrace/fs_usage even as root. If fs_usage itself prints nothing, fs-tracer cannot help. Use a non-platform build or consider EndpointSecurity if you need full coverage.
- **Very short-lived commands** may finish before fs_usage attaches. Workaround: wrap with `sh -c 'yourcmd; sleep 1'` to keep the PID alive briefly.
- **PID filter trade-off**: `--no-pid-filter` only disables Go-side filtering; fs_usage is still started with the target PID, so kernel tracing stays narrow. Use `--allow-process`/`--ignore-*` to shape events if you need child processes.
- **Full Disk Access**: granting FDA to Terminal/sudo generally does not affect fs_usage output; missing events are usually due to SIP or sampling, not TCC.

## Output modes
- Default: unique, sorted path list (text or JSON array with `--json`)
- `--events`: chronological event lines (or JSON lines with `--json`)
- `--split-access`: read vs write sets (text sections or JSON object)
- `--sandbox-snippet`: s-expressions for sandbox-exec (read/write separated when `--split-access`)

## Development
```
go test ./...
```
Pure functions are isolated under `internal/` for fs_usage parsing, filtering/classification, and output generation. CLI parsing is handled by cobra in `cmd/fs-tracer`.
