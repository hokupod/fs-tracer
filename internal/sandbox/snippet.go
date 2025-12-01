package sandbox

import (
	"bytes"
	"sort"
	"strings"
)

// BuildSnippets converts read/write path sets into sandbox-exec S expressions.
func BuildSnippets(reads, writes []string) string {
	var buf bytes.Buffer
	if len(reads) > 0 {
		writeBlock(&buf, "file-read*", reads)
	}
	if len(writes) > 0 {
		if buf.Len() > 0 {
			buf.WriteByte('\n')
		}
		writeBlock(&buf, "file-write*", writes)
	}
	return strings.TrimSpace(buf.String())
}

func writeBlock(buf *bytes.Buffer, perm string, paths []string) {
	sorted := append([]string(nil), paths...)
	sort.Strings(sorted)
	buf.WriteString("(allow ")
	buf.WriteString(perm)
	buf.WriteByte('\n')
	for _, p := range sorted {
		buf.WriteString("  (literal \"")
		buf.WriteString(escapeLiteral(p))
		buf.WriteString("\")\n")
	}
	buf.WriteString(")\n")
}

func escapeLiteral(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}
