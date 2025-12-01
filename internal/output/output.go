package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hokupod/fs-tracer/internal/fsusage"
)

// EventLine renders a single event in text mode.
func EventLine(ev fsusage.Event) string {
	ts := formatTimestamp(ev)
	return fmt.Sprintf("[%s] pid=%d comm=%s op=%s path=%q", ts, ev.PID, ev.Comm, ev.Op, ev.Path)
}

// EventsJSONLines renders events as one JSON object per line.
func EventsJSONLines(events []fsusage.Event) ([]string, error) {
	lines := make([]string, 0, len(events))
	for _, ev := range events {
		payload := map[string]interface{}{
			"timestamp": formatTimestamp(ev),
			"pid":       ev.PID,
			"comm":      ev.Comm,
			"op":        ev.Op,
			"path":      ev.Path,
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		lines = append(lines, string(b))
	}
	return lines, nil
}

// PathsText joins paths with newline.
func PathsText(paths []string) string {
	return strings.Join(paths, "\n")
}

// SplitAccessText renders read/write sections.
func SplitAccessText(reads, writes []string) string {
	var buf bytes.Buffer
	buf.WriteString("# READ\n")
	for i, p := range reads {
		buf.WriteString(p)
		if i != len(reads)-1 {
			buf.WriteByte('\n')
		}
	}
	buf.WriteString("\n\n# WRITE\n")
	for i, p := range writes {
		buf.WriteString(p)
		if i != len(writes)-1 {
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

// PathsJSON marshals path list into JSON array.
func PathsJSON(paths []string) ([]byte, error) {
	return json.Marshal(paths)
}

func formatTimestamp(ev fsusage.Event) string {
	if !ev.Timestamp.IsZero() {
		return ev.Timestamp.Format("2006-01-02T15:04:05.000")
	}
	if ev.RawTimestamp != "" {
		return ev.RawTimestamp
	}
	return ""
}
