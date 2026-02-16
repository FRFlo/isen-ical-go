// Package ical provides iCal generation functionality compliant with RFC 5545
package ical

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/FRFlo/isen-ical-go/internal/models"
)

// GenerateICal generates an iCal calendar from a slice of AurionEvents
func GenerateICal(events []models.AurionEvent) string {
	var lines []string

	// VCALENDAR header
	lines = append(lines, "BEGIN:VCALENDAR")
	lines = append(lines, "VERSION:2.0")
	lines = append(lines, "PRODID:-//ISEN-ICAL//isen-ical//EN")
	lines = append(lines, "CALSCALE:GREGORIAN")
	lines = append(lines, "METHOD:PUBLISH")
	lines = append(lines, "X-WR-TIMEZONE:Europe/Paris")

	// VTIMEZONE definition for Europe/Paris
	lines = append(lines, "BEGIN:VTIMEZONE")
	lines = append(lines, "TZID:Europe/Paris")
	lines = append(lines, "BEGIN:DAYLIGHT")
	lines = append(lines, "TZOFFSETFROM:+0100")
	lines = append(lines, "TZOFFSETTO:+0200")
	lines = append(lines, "TZNAME:CEST")
	lines = append(lines, "DTSTART:19700329T020000")
	lines = append(lines, "RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=-1SU")
	lines = append(lines, "END:DAYLIGHT")
	lines = append(lines, "BEGIN:STANDARD")
	lines = append(lines, "TZOFFSETFROM:+0200")
	lines = append(lines, "TZOFFSETTO:+0100")
	lines = append(lines, "TZNAME:CET")
	lines = append(lines, "DTSTART:19701025T030000")
	lines = append(lines, "RRULE:FREQ=YEARLY;BYMONTH=10;BYDAY=-1SU")
	lines = append(lines, "END:STANDARD")
	lines = append(lines, "END:VTIMEZONE")

	// Generate VEVENT for each event
	for _, event := range events {
		vevent := generateVEvent(event)
		lines = append(lines, vevent...)
	}

	// VCALENDAR footer
	lines = append(lines, "END:VCALENDAR")

	// Fold all lines and join with CRLF
	var foldedLines []string
	for _, line := range lines {
		folded := foldLine(line)
		foldedParts := strings.Split(folded, "\r\n")
		foldedLines = append(foldedLines, foldedParts...)
	}

	return strings.Join(foldedLines, "\r\n")
}

// generateVEvent generates a VEVENT component from an AurionEvent
func generateVEvent(event models.AurionEvent) []string {
	location, additionalInfo, subject, courseType, professor := parseTitle(event.Title)

	summary := buildSummary(subject, courseType, location, additionalInfo)
	description := buildDescription(additionalInfo, professor, courseType)

	var lines []string
	lines = append(lines, "BEGIN:VEVENT")
	lines = append(lines, fmt.Sprintf("UID:%s@isen-ical", event.ID))
	lines = append(lines, fmt.Sprintf("DTSTAMP:%s", formatDateUTC(time.Now())))
	lines = append(lines, fmt.Sprintf("DTSTART:%s", formatDateUTC(parseDate(event.Start))))
	lines = append(lines, fmt.Sprintf("DTEND:%s", formatDateUTC(parseDate(event.End))))
	lines = append(lines, fmt.Sprintf("SUMMARY:%s", escapeText(summary)))

	if location != "" {
		lines = append(lines, fmt.Sprintf("LOCATION:%s", escapeText(normalizeSpaces(location))))
	}

	if description != "" {
		lines = append(lines, fmt.Sprintf("DESCRIPTION:%s", escapeText(description)))
	}

	lines = append(lines, "END:VEVENT")

	return lines
}

// parseTitle parses the 5-line Aurion title format
// Format: "Location\nAdditionalInfo\nSubject\nCourseType\nProfessor"
// All fields are optional and may be empty
func parseTitle(title string) (location, additionalInfo, subject, courseType, professor string) {
	parts := strings.Split(title, "\n")

	// Pad with empty strings if less than 5 parts
	for len(parts) < 5 {
		parts = append(parts, "")
	}

	return strings.TrimSpace(parts[0]),
		strings.TrimSpace(parts[1]),
		strings.TrimSpace(parts[2]),
		strings.TrimSpace(parts[3]),
		strings.TrimSpace(parts[4])
}

// buildSummary builds the event summary with emoji prefix
func buildSummary(subject, courseType, location, additionalInfo string) string {
	var summary string

	// Priority: subject > courseType > location > additionalInfo > default
	switch {
	case subject != "":
		summary = subject
	case courseType != "":
		summary = courseType
	case location != "":
		summary = location
	case additionalInfo != "":
		summary = additionalInfo
	default:
		summary = "Événement sans titre"
	}

	// Add emoji prefix based on course type
	switch courseType {
	case "EXAM_SURV":
		summary = "🎓 " + summary
	case "AUTO_APPR":
		summary = "🏠 " + summary
	}

	return normalizeSpaces(summary)
}

// buildDescription builds the event description
func buildDescription(additionalInfo, professor, courseType string) string {
	var parts []string

	if additionalInfo != "" {
		parts = append(parts, additionalInfo)
		parts = append(parts, "") // Empty line separator
	}

	if professor != "" {
		parts = append(parts, fmt.Sprintf("Professeur: %s", professor))
	}

	if courseType != "" {
		parts = append(parts, fmt.Sprintf("Type de cours: %s", courseType))
	}

	return strings.Join(parts, "\n")
}

// escapeText escapes special characters in iCal text values
// RFC 5545 requires escaping of: backslash, comma, semicolon, newline
func escapeText(text string) string {
	// Order matters: escape backslash first
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}

// foldLine folds a line at 75 bytes according to RFC 5545
// Continuation lines start with a space
func foldLine(line string) string {
	const maxLength = 75
	const continuationLength = 74

	// Calculate byte length (UTF-8)
	byteLength := len(line)
	if byteLength <= maxLength {
		return line
	}

	var chunks []string
	currentPos := 0
	bytes := []byte(line)

	for currentPos < len(bytes) {
		remaining := len(bytes) - currentPos
		chunkSize := maxLength
		if len(chunks) > 0 {
			chunkSize = continuationLength
		}

		if remaining <= chunkSize {
			chunks = append(chunks, string(bytes[currentPos:]))
			break
		}

		// Find a safe position to split (don't split in the middle of a UTF-8 sequence)
		endPos := currentPos + chunkSize
		for endPos > currentPos && !utf8.RuneStart(bytes[endPos]) {
			endPos--
		}

		// If we can't find a valid split point, just use the original position
		if endPos == currentPos {
			endPos = currentPos + chunkSize
		}

		chunks = append(chunks, string(bytes[currentPos:endPos]))
		currentPos = endPos
	}

	// Join with CRLF followed by space for continuation
	return strings.Join(chunks, "\r\n ")
}

// formatDateUTC formats a time.Time as UTC iCal timestamp (YYYYMMDDTHHMMSSZ)
func formatDateUTC(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// parseDate parses a date string (timestamp or ISO format) into time.Time
func parseDate(dateValue string) time.Time {
	// Try parsing as Unix timestamp (milliseconds)
	if ms, err := strconv.ParseInt(dateValue, 10, 64); err == nil {
		return time.Unix(ms/1000, 0)
	}

	// Try parsing as ISO 8601 format
	if t, err := time.Parse(time.RFC3339, dateValue); err == nil {
		return t
	}

	// Fallback to standard parsing
	return time.Now()
}

// normalizeSpaces normalizes whitespace in text
func normalizeSpaces(text string) string {
	// Replace multiple spaces/tabs with single space
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}
