package output

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/hokupod/fs-tracer/internal/fsusage"
)

func sampleEvent() fsusage.Event {
	return fsusage.Event{
		Timestamp: time.Date(2025, time.November, 29, 10, 12, 33, 123000000, time.Local),
		PID:       1234,
		Comm:      "mytool",
		Op:        "open",
		Path:      "/etc/hosts",
	}
}

func TestEventLine(t *testing.T) {
	line := EventLine(sampleEvent())
	want := `[2025-11-29T10:12:33.123] pid=1234 comm=mytool op=open path="/etc/hosts"`
	if line != want {
		t.Fatalf("got %q want %q", line, want)
	}
}

func TestEventsJSONLines(t *testing.T) {
	ev := sampleEvent()
	lines, err := EventsJSONLines([]fsusage.Event{ev})
	if err != nil {
		t.Fatalf("EventsJSONLines error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var obj map[string]interface{}
	if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if obj["path"] != "/etc/hosts" || obj["comm"] != "mytool" || obj["op"] != "open" {
		t.Fatalf("unexpected json content: %v", obj)
	}
}

func TestPathsText(t *testing.T) {
	out := PathsText([]string{"/a", "/b"})
	if out != "/a\n/b" {
		t.Fatalf("unexpected paths text: %q", out)
	}
}

func TestSplitAccessText(t *testing.T) {
	text := SplitAccessText([]string{"/r1"}, []string{"/w1"})
	if !strings.Contains(text, "# READ") || !strings.Contains(text, "# WRITE") {
		t.Fatalf("section headers missing: %q", text)
	}
	if !strings.Contains(text, "/r1") || !strings.Contains(text, "/w1") {
		t.Fatalf("paths missing: %q", text)
	}
}

func TestPathsJSON(t *testing.T) {
	data, err := PathsJSON([]string{"/a", "/b"})
	if err != nil {
		t.Fatalf("PathsJSON error: %v", err)
	}
	var arr []string
	if err := json.Unmarshal(data, &arr); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(arr) != 2 || arr[0] != "/a" || arr[1] != "/b" {
		t.Fatalf("unexpected array: %v", arr)
	}
}
