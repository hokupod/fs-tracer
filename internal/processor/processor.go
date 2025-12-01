package processor

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/hokupod/fs-tracer/internal/fsusage"
)

// Filters represents ignore rules for events.
type Filters struct {
	AllowProcesses  []string
	IgnoreProcesses []string
	IgnorePrefixes  []string
	MaxDepth        int
	Raw             bool
}

// ApplyFilters removes events matching ignore rules unless Raw is true.
func ApplyFilters(events []fsusage.Event, f Filters) []fsusage.Event {
	if f.Raw {
		return append([]fsusage.Event(nil), events...)
	}
	var out []fsusage.Event
	for _, ev := range events {
		if len(f.AllowProcesses) > 0 && !contains(f.AllowProcesses, ev.Comm) {
			continue
		}
		if contains(f.IgnoreProcesses, ev.Comm) {
			continue
		}
		path := truncateDepth(ev.Path, f.MaxDepth)
		if hasPrefix(path, f.IgnorePrefixes) {
			continue
		}
		if path != ev.Path {
			ev.Path = path
		}
		out = append(out, ev)
	}
	return out
}

// UniqueSortedPaths collects unique paths (or their parent directories) and returns them sorted.
func UniqueSortedPaths(events []fsusage.Event, dirsOnly bool) []string {
	set := map[string]struct{}{}
	for _, ev := range events {
		p := normalizePath(ev.Path, dirsOnly)
		set[p] = struct{}{}
	}
	return toSortedSlice(set)
}

// ClassifyPaths separates paths into read and write sets based on operation kind.
func ClassifyPaths(events []fsusage.Event, dirsOnly bool) (reads []string, writes []string) {
	readSet := map[string]struct{}{}
	writeSet := map[string]struct{}{}
	for _, ev := range events {
		p := normalizePath(ev.Path, dirsOnly)
		if isWriteOp(ev.Op) {
			writeSet[p] = struct{}{}
		} else {
			readSet[p] = struct{}{}
		}
	}
	return toSortedSlice(readSet), toSortedSlice(writeSet)
}

func normalizePath(p string, dirsOnly bool) string {
	if !dirsOnly {
		return p
	}
	dir := filepath.Dir(p)
	if dir == "." {
		return p
	}
	return dir
}

func contains(list []string, target string) bool {
	for _, v := range list {
		if v == target {
			return true
		}
	}
	return false
}

func hasPrefix(path string, prefixes []string) bool {
	for _, pre := range prefixes {
		if strings.HasPrefix(path, pre) {
			return true
		}
	}
	return false
}

func isWriteOp(op string) bool {
	lo := strings.ToLower(op)
	if strings.Contains(lo, "write") || strings.HasPrefix(lo, "wr") {
		return true
	}
	switch lo {
	case "rename", "unlink", "link", "symlink", "mkdir", "rmdir", "removefile", "create",
		"fsync", "truncate", "ftruncate", "chown", "chmod", "setattrlist":
		return true
	default:
		return false
	}
}

func truncateDepth(path string, maxDepth int) string {
	if maxDepth <= 0 {
		return path
	}
	if path == "" || path == "/" {
		return path
	}
	parts := strings.Split(path, "/")
	// Handle leading slash
	start := 0
	if parts[0] == "" {
		start = 1
	}
	if len(parts)-start <= maxDepth {
		return path
	}
	trimmed := "/" + strings.Join(parts[start:start+maxDepth], "/")
	return trimmed
}

func toSortedSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
