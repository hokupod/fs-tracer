package fsusage

import (
	"testing"
	"time"
)

func baseDate() time.Time {
	return time.Date(2025, time.November, 29, 0, 0, 0, 0, time.Local)
}

func TestParseLineBasic(t *testing.T) {
	line := "22:53:18.123456 open F=3 /etc/hosts 0.000015 mytool.1234"
	ev, err := ParseLine(line, baseDate())
	if err != nil {
		t.Fatalf("ParseLine error: %v", err)
	}
	if ev.Op != "open" || ev.Path != "/etc/hosts" || ev.PID != 1234 || ev.Comm != "mytool" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	wantTs := time.Date(2025, time.November, 29, 22, 53, 18, 123456000, time.Local)
	if !ev.Timestamp.Equal(wantTs) {
		t.Fatalf("timestamp mismatch: got %v want %v", ev.Timestamp, wantTs)
	}
}

func TestParseLineWithSpacesAndNoDelta(t *testing.T) {
	line := "07:01:02.003 stat64 F=3 (R,E) /Users/testuser/Library/Application Support/App/config finder.5678"
	ev, err := ParseLine(line, baseDate())
	if err != nil {
		t.Fatalf("ParseLine error: %v", err)
	}
	if ev.Path != "/Users/testuser/Library/Application Support/App/config" {
		t.Fatalf("path mismatch: %q", ev.Path)
	}
	if ev.PID != 5678 || ev.Comm != "finder" {
		t.Fatalf("proc mismatch: %+v", ev)
	}
}

func TestParseLineInvalidProcess(t *testing.T) {
	line := "10:00:00.000 open /tmp/foo someproc-no-pid"
	if _, err := ParseLine(line, baseDate()); err == nil {
		t.Fatalf("expected error for invalid process field")
	}
}
