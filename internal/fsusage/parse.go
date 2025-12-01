package fsusage

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Event represents a single fs_usage line after parsing.
type Event struct {
	Timestamp    time.Time
	RawTimestamp string
	PID          int
	Comm         string
	Op           string
	Path         string
}

var procRe = regexp.MustCompile(`^(.*)\.(\d+)$`)

// ParseLine parses a fs_usage log line into an Event. baseDate supplies the date
// component because fs_usage outputs only time-of-day.
func ParseLine(line string, baseDate time.Time) (Event, error) {
	fields := strings.Fields(line)
	if len(fields) < 4 {
		return Event{}, fmt.Errorf("invalid fs_usage line: %q", line)
	}

	tsToken := fields[0]
	op := fields[1]

	processField := fields[len(fields)-1]
	pathEnd := len(fields) - 2

	// Trim trailing 'W' and duration tokens.
	if pathEnd >= 0 && fields[pathEnd] == "W" {
		pathEnd--
	}
	if pathEnd >= 0 && looksLikeDuration(fields[pathEnd]) {
		pathEnd--
	}

	if pathEnd < 2 {
		return Event{}, fmt.Errorf("no path/duration area in line: %q", line)
	}

	// Find first token that looks like a path.
	pathStart := -1
	for i := 2; i <= pathEnd; i++ {
		if strings.HasPrefix(fields[i], "/") {
			pathStart = i
			break
		}
	}
	if pathStart == -1 {
		return Event{}, fmt.Errorf("no path found in line: %q", line)
	}
	path := strings.Join(fields[pathStart:pathEnd+1], " ")

	// Parse process field to command and pid.
	m := procRe.FindStringSubmatch(processField)
	if len(m) != 3 {
		return Event{}, errors.New("invalid process field")
	}
	pid, err := strconv.Atoi(m[2])
	if err != nil {
		return Event{}, fmt.Errorf("invalid pid: %w", err)
	}
	comm := m[1]

	return Event{
		Timestamp:    buildTimestamp(tsToken, baseDate),
		RawTimestamp: tsToken,
		PID:          pid,
		Comm:         comm,
		Op:           strings.ToLower(op),
		Path:         path,
	}, nil
}

func buildTimestamp(token string, baseDate time.Time) time.Time {
	layouts := []string{
		"15:04:05.000000000",
		"15:04:05.000000",
		"15:04:05.00000",
		"15:04:05.0000",
		"15:04:05.000",
		"15:04:05.00",
		"15:04:05.0",
		"15:04:05",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, token, baseDate.Location()); err == nil {
			return time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
				t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), baseDate.Location())
		}
	}
	return time.Time{}
}

func looksLikeDuration(tok string) bool {
	if len(tok) == 0 {
		return false
	}
	if _, err := strconv.ParseFloat(tok, 64); err == nil {
		return true
	}
	return false
}
