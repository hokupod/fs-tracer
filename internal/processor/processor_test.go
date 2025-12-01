package processor

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/hokupod/fs-tracer/internal/fsusage"
)

func sampleEvents() []fsusage.Event {
	ts := time.Now()
	return []fsusage.Event{
		{Timestamp: ts, Comm: "mytool", Op: "open", Path: "/etc/hosts"},
		{Timestamp: ts, Comm: "mytool", Op: "write", Path: "/tmp/out.log"},
		{Timestamp: ts, Comm: "trustd", Op: "open", Path: "/System/Library/abc"},
	}
}

func TestApplyFilters(t *testing.T) {
	evs := sampleEvents()
	filtered := ApplyFilters(evs, Filters{
		IgnoreProcesses: []string{"trustd"},
		IgnorePrefixes:  []string{"/System"},
	})
	if len(filtered) != 2 {
		t.Fatalf("expected 2 events, got %d", len(filtered))
	}
	for _, ev := range filtered {
		if ev.Comm == "trustd" {
			t.Fatalf("trustd should be filtered out")
		}
		if ev.Path == "/System/Library/abc" {
			t.Fatalf("path with /System prefix should be filtered out")
		}
	}
}

func TestApplyFiltersAllowList(t *testing.T) {
	evs := sampleEvents()
	filtered := ApplyFilters(evs, Filters{
		AllowProcesses: []string{"mytool"},
	})
	if len(filtered) != 2 {
		t.Fatalf("expected only mytool events, got %d", len(filtered))
	}
	for _, ev := range filtered {
		if ev.Comm != "mytool" {
			t.Fatalf("unexpected comm %s", ev.Comm)
		}
	}
}

func TestApplyFiltersRawSkips(t *testing.T) {
	evs := sampleEvents()
	filtered := ApplyFilters(evs, Filters{Raw: true, IgnoreProcesses: []string{"trustd"}, IgnorePrefixes: []string{"/System"}})
	if !reflect.DeepEqual(filtered, evs) {
		t.Fatalf("raw mode should bypass filters")
	}
}

func TestUniqueSortedPaths(t *testing.T) {
	evs := sampleEvents()
	paths := UniqueSortedPaths(evs, false)
	want := []string{"/System/Library/abc", "/etc/hosts", "/tmp/out.log"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths mismatch: got %v want %v", paths, want)
	}
}

func TestUniqueSortedPathsDirsOnly(t *testing.T) {
	evs := sampleEvents()
	paths := UniqueSortedPaths(evs, true)
	want := []string{"/System/Library", "/etc", "/tmp"}
	if !reflect.DeepEqual(paths, want) {
		t.Fatalf("dirs mismatch: got %v want %v", paths, want)
	}
}

func TestClassifyPaths(t *testing.T) {
	evs := sampleEvents()
	read, write := ClassifyPaths(evs, false)
	sort.Strings(read)
	sort.Strings(write)
	if !reflect.DeepEqual(read, []string{"/System/Library/abc", "/etc/hosts"}) {
		t.Fatalf("read mismatch: %v", read)
	}
	if !reflect.DeepEqual(write, []string{"/tmp/out.log"}) {
		t.Fatalf("write mismatch: %v", write)
	}
}

func TestTruncateDepth(t *testing.T) {
	tests := []struct {
		path     string
		maxDepth int
		want     string
	}{
		{"/a/b/c/d", 2, "/a/b"},
		{"/a/b", 2, "/a/b"},
		{"/a/b/c", 0, "/a/b/c"},
		{"relative/path", 1, "/relative"},
		{"", 2, ""},
		{"/", 2, "/"},
	}
	for _, tt := range tests {
		if got := truncateDepth(tt.path, tt.maxDepth); got != tt.want {
			t.Fatalf("truncateDepth(%q, %d) = %q want %q", tt.path, tt.maxDepth, got, tt.want)
		}
	}
}
