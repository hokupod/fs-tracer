package sandbox

import (
	"strings"
	"testing"
)

func TestBuildSnippets(t *testing.T) {
	read := []string{"/etc/hosts", "/etc/resolv.conf"}
	write := []string{"/tmp/out.log"}
	out := BuildSnippets(read, write)
	if !containsAll(out, []string{"file-read*", "(literal \"/etc/hosts\")", "(literal \"/etc/resolv.conf\")", "file-write*", "(literal \"/tmp/out.log\")"}) {
		t.Fatalf("snippet missing expected content:\n%s", out)
	}
}

func TestBuildSnippetsReadOnly(t *testing.T) {
	out := BuildSnippets([]string{"/a"}, nil)
	if !containsAll(out, []string{"file-read*", "(literal \"/a\")"}) {
		t.Fatalf("read-only snippet incorrect: %s", out)
	}
	if strings.Contains(out, "file-write*") {
		t.Fatalf("write block should not appear: %s", out)
	}
}

func containsAll(s string, subs []string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
