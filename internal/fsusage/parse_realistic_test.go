package fsusage

import (
	"testing"
	"time"
)

func TestParseLineRealFsUsage(t *testing.T) {
	line := "00:02:07.151327    RdData[S]       D=0x07c753d6  B=0x1000   /dev/disk3s1  /Users/testuser/Library/Application Support/app/Profiles/default/AlternateServices.bin                                             0.000952 W zen.2487526"
	ev, err := ParseLine(line, baseDate2())
	if err != nil {
		t.Fatalf("ParseLine error: %v", err)
	}
	if ev.PID != 2487526 || ev.Comm != "zen" {
		t.Fatalf("proc parse mismatch: %+v", ev)
	}
	if ev.Path != "/dev/disk3s1 /Users/testuser/Library/Application Support/app/Profiles/default/AlternateServices.bin" {
		t.Fatalf("path mismatch: %q", ev.Path)
	}
	if ev.Op == "" {
		t.Fatalf("op should not be empty")
	}
}

func baseDate2() time.Time {
	return time.Date(2025, time.November, 29, 0, 0, 0, 0, time.Local)
}
