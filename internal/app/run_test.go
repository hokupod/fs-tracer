package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hokupod/fs-tracer/internal/args"
	"github.com/hokupod/fs-tracer/internal/output"
)

type fakeRunner struct {
	data string
}

func (f fakeRunner) Run(pid int, comm string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader(f.data)), nil
}

type templRunner struct {
	template string
}

func (t templRunner) Run(pid int, comm string) (io.ReadCloser, error) {
	data := fmt.Sprintf(t.template, pid, pid+1, pid+999)
	return io.NopCloser(strings.NewReader(data)), nil
}

func noopBuilder(argv []string) (*exec.Cmd, error) {
	return exec.Command("true"), nil
}

func baseDate() time.Time {
	return time.Date(2025, time.November, 29, 0, 0, 0, 0, time.Local)
}

func commandArgs() []string {
	return []string{"sh", "-c", "true"}
}

func TestRunDefaultOutput(t *testing.T) {
	opts := args.Options{Command: commandArgs()}
	log := "10:00:00.000 open /etc/hosts 0.0001 mytool.1\n10:00:00.050 write /tmp/out 0.0001 mytool.1\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:          opts,
		Runner:           fakeRunner{data: log},
		Stdout:           &out,
		Stderr:           &bytes.Buffer{},
		BaseDate:         baseDate,
		EnsureSudo:       func(bool) error { return nil },
		DisablePIDFilter: true,
		CmdBuilder:       noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	got := out.String()
	want := output.HeaderLine() + "\n" + "/etc/hosts\n/tmp/out\n"
	if got != want {
		t.Fatalf("output mismatch:\n%s\nwant:\n%s", got, want)
	}
}

func TestRunSplitAccessJSON(t *testing.T) {
	opts := args.Options{Command: commandArgs(), JSON: true, SplitAccess: true}
	log := "10:00:00.000 open /etc/hosts 0.0001 mytool.1\n10:00:00.050 write /tmp/out 0.0001 mytool.1\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:          opts,
		Runner:           fakeRunner{data: log},
		Stdout:           &out,
		Stderr:           &bytes.Buffer{},
		BaseDate:         baseDate,
		EnsureSudo:       func(bool) error { return nil },
		DisablePIDFilter: true,
		CmdBuilder:       noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	var obj map[string][]string
	if err := json.Unmarshal(bytes.TrimSpace(out.Bytes()), &obj); err != nil {
		t.Fatalf("json parse error: %v", err)
	}
	if len(obj["read"]) != 1 || obj["read"][0] != "/etc/hosts" {
		t.Fatalf("read set mismatch: %v", obj["read"])
	}
	if len(obj["write"]) != 1 || obj["write"][0] != "/tmp/out" {
		t.Fatalf("write set mismatch: %v", obj["write"])
	}
}

func TestRunEventsJSON(t *testing.T) {
	opts := args.Options{Command: commandArgs(), JSON: true, Events: true}
	log := "10:00:00.000 open /etc/hosts 0.0001 mytool.1\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:          opts,
		Runner:           fakeRunner{data: log},
		Stdout:           &out,
		Stderr:           &bytes.Buffer{},
		BaseDate:         baseDate,
		EnsureSudo:       func(bool) error { return nil },
		DisablePIDFilter: true,
		CmdBuilder:       noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	if obj["path"] != "/etc/hosts" {
		t.Fatalf("path mismatch: %v", obj["path"])
	}
}

func TestRunEventsTextHasHeader(t *testing.T) {
	opts := args.Options{Command: commandArgs(), Events: true}
	log := "10:00:00.000 open /etc/hosts 0.0001 mytool.1\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:          opts,
		Runner:           fakeRunner{data: log},
		Stdout:           &out,
		Stderr:           &bytes.Buffer{},
		BaseDate:         baseDate,
		EnsureSudo:       func(bool) error { return nil },
		DisablePIDFilter: true,
		CmdBuilder:       noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header and at least one event, got %v", lines)
	}
	if lines[0] != output.HeaderLine() {
		t.Fatalf("header mismatch: %q", lines[0])
	}
	if !strings.Contains(lines[1], "/etc/hosts") {
		t.Fatalf("event line missing path: %q", lines[1])
	}
}

func TestRunSandboxSnippet(t *testing.T) {
	opts := args.Options{Command: commandArgs(), SandboxSnippet: true}
	log := "10:00:00.000 open /etc/hosts 0.0001 mytool.1\n10:00:00.050 write /tmp/out 0.0001 mytool.1\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:          opts,
		Runner:           fakeRunner{data: log},
		Stdout:           &out,
		Stderr:           &bytes.Buffer{},
		BaseDate:         baseDate,
		EnsureSudo:       func(bool) error { return nil },
		DisablePIDFilter: true,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	s := out.String()
	if !strings.HasPrefix(s, output.HeaderLine()) {
		t.Fatalf("header missing: %q", s)
	}
	if !strings.Contains(s, "file-read*") || !strings.Contains(s, "file-write*") {
		t.Fatalf("snippet missing sections: %s", s)
	}
}

func TestRunFollowChildrenFiltersOtherPIDs(t *testing.T) {
	opts := args.Options{Command: commandArgs(), FollowChildren: true}
	logTemplate := "10:00:00.000 open /parent/file 0.0001 parent.%d\n" +
		"10:00:00.010 open /child/file 0.0001 child.%d\n" +
		"10:00:00.020 open /other/file 0.0001 other.%d\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:    opts,
		Runner:     templRunner{template: logTemplate},
		Stdout:     &out,
		Stderr:     &bytes.Buffer{},
		BaseDate:   baseDate,
		EnsureSudo: func(bool) error { return nil },
		ChildFinder: func(root int) ([]int, error) {
			return []int{root + 1}, nil
		},
		ThreadLister: func(pid int) ([]uint64, error) {
			return []uint64{uint64(pid)}, nil
		},
		CmdBuilder: noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	expectedBody := "/child/file\n/parent/file\n"
	expected := output.HeaderLine() + "\n" + expectedBody
	if out.String() != expected {
		t.Fatalf("output mismatch:\n%s\nwant:\n%s", out.String(), expected)
	}
}

func TestRunFollowChildrenAllowsThreadIDs(t *testing.T) {
	opts := args.Options{Command: commandArgs(), FollowChildren: true}
	logTemplate := "10:00:00.000 open /tmp/root 0.0001 root.%d\n" +
		"10:00:00.010 open /tmp/child 0.0001 child.%d\n" +
		"10:00:00.020 open /tmp/thread 0.0001 root.%d\n"
	var out bytes.Buffer
	threadLister := func(pid int) ([]uint64, error) {
		// Allow pid itself and a synthetic thread id pid+999 to simulate fs_usage TID。
		return []uint64{uint64(pid), uint64(pid + 999)}, nil
	}
	code := Run(Config{
		Options:      opts,
		Runner:       templRunner{template: logTemplate},
		Stdout:       &out,
		Stderr:       &bytes.Buffer{},
		BaseDate:     baseDate,
		EnsureSudo:   func(bool) error { return nil },
		ChildFinder:  func(int) ([]int, error) { return nil, nil },
		ThreadLister: threadLister,
		CommFinder:   func(int) (string, error) { return "root", nil },
		CmdBuilder:   noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected header plus paths, got: %v", lines)
	}
	paths := lines[1:]
	if !contains(paths, "/tmp/thread") {
		t.Fatalf("thread-id path missing: %v", paths)
	}
}

func TestRunFollowChildrenThreadLookupFailureDisablesFilter(t *testing.T) {
	opts := args.Options{Command: commandArgs(), FollowChildren: true}
	logTemplate := "10:00:00.000 open /tmp/thread 0.0001 root.%d\n" +
		"10:00:00.010 open /tmp/tid 0.0001 root.%d\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:    opts,
		Runner:     templRunner{template: logTemplate},
		Stdout:     &out,
		Stderr:     &bytes.Buffer{},
		BaseDate:   baseDate,
		EnsureSudo: func(bool) error { return nil },
		ChildFinder: func(int) ([]int, error) {
			return nil, nil
		},
		ThreadLister: func(int) ([]uint64, error) {
			return nil, syscall.EPERM
		},
		CommFinder: func(pid int) (string, error) {
			return "root", nil
		},
		CmdBuilder: noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(out.String(), "/tmp/tid") {
		t.Fatalf("expected output despite thread lookup failure, got: %q", out.String())
	}
}

func TestRunFollowChildrenZeroMatchBypass(t *testing.T) {
	opts := args.Options{Command: commandArgs(), FollowChildren: true}
	var b strings.Builder
	for i := 0; i < 50; i++ {
		fmt.Fprintf(&b, "10:00:00.%03d open /tmp/file%02d 0.0001 root.%d\n", i, i%5, i+100)
	}
	log := b.String()
	var out bytes.Buffer
	code := Run(Config{
		Options:    opts,
		Runner:     fakeRunner{data: log},
		Stdout:     &out,
		Stderr:     &bytes.Buffer{},
		BaseDate:   baseDate,
		EnsureSudo: func(bool) error { return nil },
		ChildFinder: func(int) ([]int, error) {
			return nil, nil
		},
		ThreadLister: func(int) ([]uint64, error) {
			// Return IDs that never match the log, forcing zero-match bypass.
			return []uint64{999999}, nil
		},
		CommFinder: func(int) (string, error) { return "root", nil },
		CmdBuilder: noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(out.String(), "/tmp/file") {
		t.Fatalf("expected output after bypass, got: %q", out.String())
	}
}

func TestRunPIDFilterDisabledWhenNotFollowingChildren(t *testing.T) {
	// fs_usage returns a pid (thread id) different from target; with follow-children OFF
	// we should still keep events (rely on fs_usage kernel filter).
	opts := args.Options{Command: commandArgs()}
	logTemplate := "10:00:00.000 open /tmp/file 0.0001 other.%d\n"
	var out bytes.Buffer
	code := Run(Config{
		Options:    opts,
		Runner:     templRunner{template: logTemplate},
		Stdout:     &out,
		Stderr:     &bytes.Buffer{},
		BaseDate:   baseDate,
		EnsureSudo: func(bool) error { return nil },
		CmdBuilder: noopBuilder,
	})
	if code != 0 {
		t.Fatalf("exit code = %d", code)
	}
	if !strings.Contains(out.String(), "/tmp/file") {
		t.Fatalf("event should pass without follow-children: %q", out.String())
	}
}

func TestParseDescendants(t *testing.T) {
	ps := "  PID  PPID\n  10   1\n  11   10\n  12   1\n  13   12\n"
	desc, err := parseDescendants(1, []byte(ps))
	if err != nil {
		t.Fatalf("parseDescendants error: %v", err)
	}
	got := strings.Join(toStrings(desc), ",")
	want := "10,11,12,13"
	if got != want {
		t.Fatalf("unexpected descendants: %s", got)
	}
}

func toStrings(nums []int) []string {
	out := make([]string, 0, len(nums))
	sorted := append([]int(nil), nums...)
	sort.Ints(sorted)
	for _, n := range sorted {
		out = append(out, fmt.Sprintf("%d", n))
	}
	return out
}

func contains(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}
