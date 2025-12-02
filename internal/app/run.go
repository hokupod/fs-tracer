package app

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/hokupod/fs-tracer/internal/args"
	"github.com/hokupod/fs-tracer/internal/fsusage"
	"github.com/hokupod/fs-tracer/internal/output"
	"github.com/hokupod/fs-tracer/internal/processor"
	"github.com/hokupod/fs-tracer/internal/sandbox"
)

const (
	exitInvalidArgs = 90
	exitCmdStartErr = 91
	exitFsUsageErr  = 92
	exitScanErr     = 93
)

// Config controls Run behavior; zero values pick sensible defaults.
type Config struct {
	Options          args.Options
	Runner           fsusage.FsUsageRunner
	Stdout           io.Writer
	Stderr           io.Writer
	CmdBuilder       func([]string) (*exec.Cmd, error)
	BaseDate         func() time.Time
	EnsureSudo       func(noSudo bool) error
	DisablePIDFilter bool
}

// Run executes yourcmd, collects fs_usage events, and writes output. It returns
// the intended process exit code (yourcmd or internal error in 90–99 range).
func Run(cfg Config) int {
	opts := cfg.Options

	debug := os.Getenv("FS_TRACER_DEBUG") != ""
	filterPID := !cfg.DisablePIDFilter && !opts.NoPIDFilter

	stdout := cfg.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := cfg.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	runner := cfg.Runner
	if runner == nil {
		runner = fsusage.SudoFsUsageRunner{NoSudo: opts.NoSudo}
	}
	builder := cfg.CmdBuilder
	if builder == nil {
		builder = defaultCmdBuilder
	}
	baseDate := time.Now
	if cfg.BaseDate != nil {
		baseDate = cfg.BaseDate
	}
	baseDateValue := baseDate()

	ensureSudo := cfg.EnsureSudo
	if ensureSudo == nil {
		ensureSudo = defaultEnsureSudo
	}

	if err := ensureSudo(opts.NoSudo); err != nil {
		fmt.Fprintln(stderr, "failed to refresh sudo timestamp:", err)
		return exitCmdStartErr
	}

	cmd, err := builder(opts.Command)
	if err != nil {
		fmt.Fprintln(stderr, "failed to build command:", err)
		return exitInvalidArgs
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = os.Environ()
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin

	if err := applyCredential(cmd); err != nil {
		fmt.Fprintln(stderr, err)
		return exitInvalidArgs
	}

	if err := cmd.Start(); err != nil {
		fmt.Fprintln(stderr, "failed to start yourcmd:", err)
		return exitCmdStartErr
	}

	comm := cmd.Path
	if comm == "" && len(cmd.Args) > 0 {
		comm = cmd.Args[0]
	}
	reader, err := runner.Run(cmd.Process.Pid, filepath.Base(comm))
	if err != nil {
		_ = cmd.Process.Kill()
		fmt.Fprintln(stderr, "failed to start fs_usage:", err)
		return exitFsUsageErr
	}

	eventsCh := make(chan fsusage.Event)
	scanErrCh := make(chan error, 1)

	// Collector drains events concurrently to avoid blocking fs_usage scanner.
	var (
		events        []fsusage.Event
		collectDoneCh = make(chan struct{})
	)
	go func() {
		for ev := range eventsCh {
			events = append(events, ev)
		}
		close(collectDoneCh)
	}()

	go func(r io.ReadCloser) {
		defer close(eventsCh)
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 0, 128*1024), 512*1024)
		for scanner.Scan() {
			line := scanner.Text()
			if debug {
				fmt.Fprintln(stderr, "fs_usage:", line)
			}
			ev, err := fsusage.ParseLine(line, baseDateValue)
			if err != nil {
				if debug {
					fmt.Fprintln(stderr, "parse error:", err, "line:", line)
				}
				continue
			}
			if filterPID && ev.PID != cmd.Process.Pid {
				continue
			}
			eventsCh <- ev
		}
		if err := scanner.Err(); err != nil {
			scanErrCh <- err
		}
	}(reader)

	errCmd := cmd.Wait()
	_ = reader.Close()

	// Wait for collector to finish draining events.
	<-collectDoneCh

	select {
	case scanErr := <-scanErrCh:
		if scanErr != nil {
			if !isBenignClose(scanErr) {
				fmt.Fprintln(stderr, "fs_usage read error:", scanErr)
				return exitScanErr
			}
		}
	default:
	}

	filters := processor.Filters{
		AllowProcesses:  opts.AllowProcesses,
		IgnoreProcesses: opts.IgnoreProcesses,
		IgnorePrefixes:  expandPrefixes(opts.IgnorePrefixes, opts.IgnoreCWD),
		MaxDepth:        opts.MaxDepth,
		Raw:             opts.Raw,
	}
	filtered := processor.ApplyFilters(events, filters)

	if err := render(stdout, opts, filtered); err != nil {
		fmt.Fprintln(stderr, "output error:", err)
		return exitScanErr
	}

	if debug && len(filtered) == 0 {
		fmt.Fprintln(stderr, "debug: no events after filtering")
	}

	return exitCodeFromCmd(errCmd)
}

func render(w io.Writer, opts args.Options, events []fsusage.Event) error {
	headerPrinted := false
	printHeader := func() {}
	if !opts.JSON {
		printHeader = func() {
			if headerPrinted {
				return
			}
			fmt.Fprintln(w, output.HeaderLine())
			headerPrinted = true
		}
	}

	if opts.Events {
		if opts.JSON {
			lines, err := output.EventsJSONLines(events)
			if err != nil {
				return err
			}
			for _, line := range lines {
				fmt.Fprintln(w, line)
			}
			return nil
		}
		printHeader()
		for _, ev := range events {
			fmt.Fprintln(w, output.EventLine(ev))
		}
		return nil
	}

	// Non-events output
	if opts.SandboxSnippet {
		printHeader()
		read, write := processor.ClassifyPaths(events, opts.DirsOnly)
		snippet := sandbox.BuildSnippets(read, write)
		fmt.Fprintln(w, snippet)
		return nil
	}

	if opts.SplitAccess {
		read, write := processor.ClassifyPaths(events, opts.DirsOnly)
		if opts.JSON {
			obj := map[string][]string{"read": read, "write": write}
			b, err := json.Marshal(obj)
			if err != nil {
				return err
			}
			fmt.Fprintln(w, string(b))
			return nil
		}
		printHeader()
		fmt.Fprintln(w, output.SplitAccessText(read, write))
		return nil
	}

	paths := processor.UniqueSortedPaths(events, opts.DirsOnly)
	if opts.JSON {
		b, err := output.PathsJSON(paths)
		if err != nil {
			return err
		}
		fmt.Fprintln(w, string(b))
		return nil
	}
	printHeader()
	fmt.Fprintln(w, output.PathsText(paths))
	return nil
}

func exitCodeFromCmd(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			if status.Signaled() {
				return 128 + int(status.Signal())
			}
			return status.ExitStatus()
		}
	}
	return exitCmdStartErr
}

func isBenignClose(err error) bool {
	msg := err.Error()
	return strings.Contains(msg, "file already closed") || strings.Contains(msg, "use of closed file") || errors.Is(err, io.ErrClosedPipe)
}

func expandPrefixes(prefixes []string, ignoreCwd bool) []string {
	out := make([]string, 0, len(prefixes)+1)
	cwd := ""
	if ignoreCwd {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	for _, p := range prefixes {
		if p == "." && cwd != "" {
			out = append(out, cwd)
			continue
		}
		out = append(out, p)
	}
	if ignoreCwd && cwd != "" {
		out = append(out, cwd)
	}
	return out
}

func defaultCmdBuilder(argv []string) (*exec.Cmd, error) {
	if len(argv) == 0 {
		return nil, fmt.Errorf("no command specified")
	}
	cmd := exec.Command(argv[0], argv[1:]...)
	return cmd, nil
}

func applyCredential(cmd *exec.Cmd) error {
	if os.Geteuid() != 0 {
		return nil
	}
	sudoUID := os.Getenv("SUDO_UID")
	sudoGID := os.Getenv("SUDO_GID")
	if sudoUID == "" || sudoGID == "" {
		return fmt.Errorf("running as root is unsupported without SUDO_UID/GID; yourcmd must run as original user")
	}
	uid, err := strconv.ParseUint(sudoUID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid SUDO_UID: %w", err)
	}
	gid, err := strconv.ParseUint(sudoGID, 10, 32)
	if err != nil {
		return fmt.Errorf("invalid SUDO_GID: %w", err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}
	return nil
}

func defaultEnsureSudo(noSudo bool) error {
	if noSudo || os.Geteuid() == 0 {
		return nil
	}
	cmd := exec.Command("sudo", "-v")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
