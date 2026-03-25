package ical

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/models"
)

type planningJSONPayload struct {
	Events []models.AurionEvent `json:"events"`
}

type parsedICalEvent struct {
	UID     string
	DTStart string
	DTEnd   string
	Summary string
}

func TestGenerateICal_ValidStructure(t *testing.T) {
	events := []models.AurionEvent{
		{
			ID:    "12345",
			Title: "Room 101\nAdditional info\nMath Exam\nEXAM_SURV\nDr. Smith",
			Start: "1705312800000", // 2024-01-15T10:00:00Z
			End:   "1705320000000", // 2024-01-15T12:00:00Z
		},
	}

	result := GenerateICal(events)

	// Check required VCALENDAR fields
	if !strings.Contains(result, "BEGIN:VCALENDAR") {
		t.Error("Missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(result, "END:VCALENDAR") {
		t.Error("Missing END:VCALENDAR")
	}
	if !strings.Contains(result, "VERSION:2.0") {
		t.Error("Missing VERSION:2.0")
	}
	if !strings.Contains(result, "PRODID:-//ISEN-ICAL//isen-ical//EN") {
		t.Error("Missing PRODID")
	}
	if !strings.Contains(result, "CALSCALE:GREGORIAN") {
		t.Error("Missing CALSCALE:GREGORIAN")
	}
	if !strings.Contains(result, "METHOD:PUBLISH") {
		t.Error("Missing METHOD:PUBLISH")
	}
	if !strings.Contains(result, "X-WR-CALNAME:Aurion") {
		t.Error("Missing X-WR-CALNAME:Aurion")
	}
	if !strings.Contains(result, "X-WR-TIMEZONE:Europe/Paris") {
		t.Error("Missing X-WR-TIMEZONE")
	}
	if strings.Contains(result, "BEGIN:VTIMEZONE") {
		t.Error("VTIMEZONE should not be present")
	}

	// Check VEVENT
	if !strings.Contains(result, "BEGIN:VEVENT") {
		t.Error("Missing BEGIN:VEVENT")
	}
	if !strings.Contains(result, "END:VEVENT") {
		t.Error("Missing END:VEVENT")
	}
	if !strings.Contains(result, "UID:12345@isen-ical") {
		t.Error("Missing UID")
	}
	if !strings.Contains(result, "DTSTART:") {
		t.Error("Missing DTSTART")
	}
	if !strings.Contains(result, "DTEND:") {
		t.Error("Missing DTEND")
	}
	if !strings.Contains(result, "DTSTAMP:") {
		t.Error("Missing DTSTAMP")
	}
	if !strings.Contains(result, "SUMMARY:") {
		t.Error("Missing SUMMARY")
	}
}

func TestGenerateICal_EmojiPrefixes(t *testing.T) {
	tests := []struct {
		name         string
		courseType   string
		wantContains string
	}{
		{
			name:         "EXAM_SURV has graduation emoji",
			courseType:   "EXAM_SURV",
			wantContains: "SUMMARY:🎓",
		},
		{
			name:         "AUTO_APPR has house emoji",
			courseType:   "AUTO_APPR",
			wantContains: "SUMMARY:🏠",
		},
		{
			name:         "Regular course has no emoji",
			courseType:   "CM",
			wantContains: "SUMMARY:Math",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := []models.AurionEvent{
				{
					ID:    "test",
					Title: "Room\nInfo\nMath\n" + tt.courseType + "\nProf",
					Start: "1705312800000",
					End:   "1705320000000",
				},
			}

			result := GenerateICal(events)

			if !strings.Contains(result, tt.wantContains) {
				t.Errorf("Expected result to contain %q, got:\n%s", tt.wantContains, result)
			}
		})
	}
}

func TestGenerateICal_LineFolding(t *testing.T) {
	// Create an event with a very long title that needs folding
	longTitle := "Room 101\nInfo\n" + strings.Repeat("A", 100) + "\nCM\nProfessor"

	events := []models.AurionEvent{
		{
			ID:    "fold-test",
			Title: longTitle,
			Start: "1705312800000",
			End:   "1705320000000",
		},
	}

	result := GenerateICal(events)

	// Check that lines are folded (continuation lines start with space)
	lines := strings.Split(result, "\r\n")
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(line, " ") {
			// This is a continuation line, which is correct
			continue
		}
		// Regular lines should not exceed 75 bytes
		if len(line) > 75 {
			t.Errorf("Line %d exceeds 75 bytes: %q (length: %d)", i, line, len(line))
		}
	}
}

func TestEscapeText(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "Simple text",
			expected: "Simple text",
		},
		{
			input:    "Text with backslash \\",
			expected: "Text with backslash \\\\",
		},
		{
			input:    "Text with comma, here",
			expected: "Text with comma\\, here",
		},
		{
			input:    "Text with semicolon; here",
			expected: "Text with semicolon\\; here",
		},
		{
			input:    "Text with\nnewline",
			expected: "Text with\\nnewline",
		},
		{
			input:    "Text with \\,;\\n all",
			expected: "Text with \\\\\\,\\;\\\\n all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeText(tt.input)
			if result != tt.expected {
				t.Errorf("escapeText(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFoldLine(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Short line not folded",
			input:    "SUMMARY:Short",
			expected: "SUMMARY:Short",
		},
		{
			name:     "Line exactly 75 bytes not folded",
			input:    strings.Repeat("A", 75),
			expected: strings.Repeat("A", 75),
		},
		{
			name:     "Line over 75 bytes folded",
			input:    strings.Repeat("A", 80),
			expected: strings.Repeat("A", 75) + "\r\n " + strings.Repeat("A", 5),
		},
		{
			name:     "Long line folded multiple times",
			input:    strings.Repeat("A", 150),
			expected: strings.Repeat("A", 75) + "\r\n " + strings.Repeat("A", 74) + "\r\n " + "A",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := foldLine(tt.input)
			if result != tt.expected {
				t.Errorf("foldLine() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseTitle(t *testing.T) {
	tests := []struct {
		name           string
		title          string
		wantLocation   string
		wantAdditional string
		wantSubject    string
		wantCourseType string
		wantProfessor  string
	}{
		{
			name:           "Full 5-line title",
			title:          "Room 101\nAdditional info\nMath Exam\nEXAM_SURV\nDr. Smith",
			wantLocation:   "Room 101",
			wantAdditional: "Additional info",
			wantSubject:    "Math Exam",
			wantCourseType: "EXAM_SURV",
			wantProfessor:  "Dr. Smith",
		},
		{
			name:           "Empty lines",
			title:          "\n\n\n\n",
			wantLocation:   "",
			wantAdditional: "",
			wantSubject:    "",
			wantCourseType: "",
			wantProfessor:  "",
		},
		{
			name:           "Less than 5 lines",
			title:          "Room 101\nMath",
			wantLocation:   "Room 101",
			wantAdditional: "Math",
			wantSubject:    "",
			wantCourseType: "",
			wantProfessor:  "",
		},
		{
			name:           "Whitespace trimmed",
			title:          "  Room 101  \n  Info  \n  Subject  \n  Type  \n  Prof  ",
			wantLocation:   "Room 101",
			wantAdditional: "Info",
			wantSubject:    "Subject",
			wantCourseType: "Type",
			wantProfessor:  "Prof",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			location, additional, subject, courseType, professor := parseTitle(tt.title)

			if location != tt.wantLocation {
				t.Errorf("location = %q, want %q", location, tt.wantLocation)
			}
			if additional != tt.wantAdditional {
				t.Errorf("additional = %q, want %q", additional, tt.wantAdditional)
			}
			if subject != tt.wantSubject {
				t.Errorf("subject = %q, want %q", subject, tt.wantSubject)
			}
			if courseType != tt.wantCourseType {
				t.Errorf("courseType = %q, want %q", courseType, tt.wantCourseType)
			}
			if professor != tt.wantProfessor {
				t.Errorf("professor = %q, want %q", professor, tt.wantProfessor)
			}
		})
	}
}

func TestBuildSummary(t *testing.T) {
	tests := []struct {
		name        string
		subject     string
		courseType  string
		location    string
		additional  string
		wantSummary string
	}{
		{
			name:        "Subject takes priority",
			subject:     "Math",
			courseType:  "CM",
			location:    "Room 101",
			additional:  "Info",
			wantSummary: "Math",
		},
		{
			name:        "CourseType used when no subject",
			subject:     "",
			courseType:  "CM",
			location:    "Room 101",
			additional:  "Info",
			wantSummary: "CM",
		},
		{
			name:        "Location used when no subject or courseType",
			subject:     "",
			courseType:  "",
			location:    "Room 101",
			additional:  "Info",
			wantSummary: "Room 101",
		},
		{
			name:        "AdditionalInfo used when no subject, courseType, or location",
			subject:     "",
			courseType:  "",
			location:    "",
			additional:  "Info",
			wantSummary: "Info",
		},
		{
			name:        "Default when all empty",
			subject:     "",
			courseType:  "",
			location:    "",
			additional:  "",
			wantSummary: "Événement sans titre",
		},
		{
			name:        "EXAM_SURV gets graduation emoji",
			subject:     "Math",
			courseType:  "EXAM_SURV",
			wantSummary: "🎓 Math",
		},
		{
			name:        "AUTO_APPR gets house emoji",
			subject:     "Physics",
			courseType:  "AUTO_APPR",
			wantSummary: "🏠 Physics",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSummary(tt.subject, tt.courseType, tt.location, tt.additional)
			if result != tt.wantSummary {
				t.Errorf("buildSummary() = %q, want %q", result, tt.wantSummary)
			}
		})
	}
}

func TestFormatDateUTC(t *testing.T) {
	tests := []struct {
		input    time.Time
		expected string
	}{
		{
			input:    time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			expected: "20240115T100000Z",
		},
		{
			input:    time.Date(2024, 12, 31, 23, 59, 59, 0, time.UTC),
			expected: "20241231T235959Z",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatDateUTC(tt.input)
			if result != tt.expected {
				t.Errorf("formatDateUTC() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
	}{
		{
			name:     "Unix timestamp milliseconds",
			input:    "1705312800000",
			expected: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			name:     "RFC3339 format",
			input:    "2024-01-15T10:00:00Z",
			expected: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			name:     "ISO offset without colon (+0100)",
			input:    "2026-03-09T09:05:00+0100",
			expected: time.Date(2026, 3, 9, 8, 5, 0, 0, time.UTC),
		},
		{
			name:     "ISO offset without colon (+0200)",
			input:    "2026-03-30T09:05:00+0200",
			expected: time.Date(2026, 3, 30, 7, 5, 0, 0, time.UTC),
		},
		{
			name:     "Date-only format",
			input:    "2024-01-15",
			expected: time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:     "Datetime with space separator",
			input:    "2024-01-15 10:00:00",
			expected: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		{
			name:     "Invalid date returns zero time",
			input:    "not-a-date",
			expected: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseDate(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("parseDate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNormalizeSpaces(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "  Multiple   spaces   ",
			expected: "Multiple spaces",
		},
		{
			input:    "\t\tTabs\t\tand\tspaces\t\t",
			expected: "Tabs and spaces",
		},
		{
			input:    "Already clean",
			expected: "Already clean",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeSpaces(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSpaces(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatLocation(t *testing.T) {
	tests := []struct {
		name     string
		location string
		expected string
	}{
		{
			name:     "keyword found appends address",
			location: "IC2 - Salle 12",
			expected: "IC2 - Salle 12 - " + locationAddressByKeyword["IC2"],
		},
		{
			name:     "keyword search is case insensitive",
			location: "ic2 amphi",
			expected: "ic2 amphi - " + locationAddressByKeyword["IC2"],
		},
		{
			name:     "no keyword keeps original location",
			location: "Room 101",
			expected: "Room 101",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatLocation(tt.location)
			if result != tt.expected {
				t.Errorf("formatLocation(%q) = %q, want %q", tt.location, result, tt.expected)
			}
		})
	}
}

func TestGenerateICal_LocationAddressEnrichment(t *testing.T) {
	events := []models.AurionEvent{
		{
			ID:    "location-enrichment",
			Title: "IC2\nAdditional info\nMath\nCM\nProf",
			Start: "1705312800000",
			End:   "1705320000000",
		},
	}

	result := GenerateICal(events)

	if !strings.Contains(result, "LOCATION:IC2 - "+strings.ReplaceAll(locationAddressByKeyword["IC2"], ",", "\\,")) {
		t.Fatalf("expected LOCATION to include mapped address, got:\n%s", result)
	}
}

func TestGenerateICal_MultipleEvents(t *testing.T) {
	events := []models.AurionEvent{
		{
			ID:    "1",
			Title: "Room A\nInfo\nMath\nCM\nProf1",
			Start: "1705312800000",
			End:   "1705320000000",
		},
		{
			ID:    "2",
			Title: "Room B\nInfo\nPhysics\nTD\nProf2",
			Start: "1705399200000",
			End:   "1705406400000",
		},
	}

	result := GenerateICal(events)

	// Should have 2 VEVENT blocks
	veventCount := strings.Count(result, "BEGIN:VEVENT")
	if veventCount != 2 {
		t.Errorf("Expected 2 VEVENT blocks, got %d", veventCount)
	}

	// Should have both UIDs
	if !strings.Contains(result, "UID:1@isen-ical") {
		t.Error("Missing UID for first event")
	}
	if !strings.Contains(result, "UID:2@isen-ical") {
		t.Error("Missing UID for second event")
	}
}

func TestGenerateICal_EmptyEvents(t *testing.T) {
	result := GenerateICal([]models.AurionEvent{})

	// Should still have valid VCALENDAR structure
	if !strings.Contains(result, "BEGIN:VCALENDAR") {
		t.Error("Missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(result, "END:VCALENDAR") {
		t.Error("Missing END:VCALENDAR")
	}

	// Should have VTIMEZONE but no VEVENT
	if strings.Contains(result, "BEGIN:VEVENT") {
		t.Error("Should not have VEVENT when no events provided")
	}
}

func TestGenerateICal_PlanningJSONDateParity(t *testing.T) {
	fixture, err := os.ReadFile("../../../tests/fixtures/planning.sample.json")
	if err != nil {
		t.Fatalf("failed to read planning.sample.json fixture: %v", err)
	}

	var payload planningJSONPayload
	if err := json.Unmarshal(fixture, &payload); err != nil {
		t.Fatalf("failed to unmarshal planning.sample.json fixture: %v", err)
	}

	icalContent := GenerateICal(payload.Events)
	parsedEvents := parseICalEvents(icalContent)

	if len(parsedEvents) != len(payload.Events) {
		t.Fatalf("VEVENT count mismatch: got %d, want %d", len(parsedEvents), len(payload.Events))
	}

	byUID := make(map[string]parsedICalEvent, len(parsedEvents))
	for _, evt := range parsedEvents {
		byUID[evt.UID] = evt
	}

	for _, raw := range payload.Events {
		uid := raw.ID + "@isen-ical"
		evt, ok := byUID[uid]
		if !ok {
			t.Fatalf("missing VEVENT for UID %s", uid)
		}

		start, err := time.Parse("2006-01-02T15:04:05-0700", raw.Start)
		if err != nil {
			t.Fatalf("invalid fixture start %q for event %s: %v", raw.Start, raw.ID, err)
		}
		end, err := time.Parse("2006-01-02T15:04:05-0700", raw.End)
		if err != nil {
			t.Fatalf("invalid fixture end %q for event %s: %v", raw.End, raw.ID, err)
		}

		expectedStart := formatDateUTC(start)
		expectedEnd := formatDateUTC(end)

		if evt.DTStart != expectedStart {
			t.Fatalf("DTSTART mismatch for %s: got %s, want %s", raw.ID, evt.DTStart, expectedStart)
		}
		if evt.DTEnd != expectedEnd {
			t.Fatalf("DTEND mismatch for %s: got %s, want %s", raw.ID, evt.DTEnd, expectedEnd)
		}
		if strings.TrimSpace(evt.Summary) == "" {
			t.Fatalf("empty SUMMARY for event %s", raw.ID)
		}
	}
}

func parseICalEvents(content string) []parsedICalEvent {
	rawLines := strings.Split(content, "\r\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if strings.HasPrefix(line, " ") && len(lines) > 0 {
			lines[len(lines)-1] += strings.TrimPrefix(line, " ")
			continue
		}
		lines = append(lines, line)
	}

	events := make([]parsedICalEvent, 0)
	inEvent := false
	current := parsedICalEvent{}

	for _, line := range lines {
		switch line {
		case "BEGIN:VEVENT":
			inEvent = true
			current = parsedICalEvent{}
		case "END:VEVENT":
			if inEvent {
				events = append(events, current)
			}
			inEvent = false
		default:
			if !inEvent {
				continue
			}
			switch {
			case strings.HasPrefix(line, "UID:"):
				current.UID = strings.TrimPrefix(line, "UID:")
			case strings.HasPrefix(line, "DTSTART:"):
				current.DTStart = strings.TrimPrefix(line, "DTSTART:")
			case strings.HasPrefix(line, "DTEND:"):
				current.DTEnd = strings.TrimPrefix(line, "DTEND:")
			case strings.HasPrefix(line, "SUMMARY:"):
				current.Summary = strings.TrimPrefix(line, "SUMMARY:")
			}
		}
	}

	return events
}
