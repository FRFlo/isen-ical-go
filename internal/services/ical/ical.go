// Package ical fournit la génération de calendriers iCal conforme à la RFC 5545
package ical

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/models"
)

// GenerateICal génère un calendrier iCal à partir d'une liste d'événements Aurion
func GenerateICal(events []models.AurionEvent) string {
	var lines []string

	// VCALENDAR header
	lines = append(lines, "BEGIN:VCALENDAR")
	lines = append(lines, "VERSION:2.0")
	lines = append(lines, "PRODID:-//ISEN-ICAL//isen-ical//EN")
	lines = append(lines, "CALSCALE:GREGORIAN")
	lines = append(lines, "METHOD:PUBLISH")
	lines = append(lines, "X-WR-CALNAME:Aurion")
	lines = append(lines, "X-WR-TIMEZONE:Europe/Paris")

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

// generateVEvent génère un composant VEVENT à partir d'un événement Aurion
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

// parseTitle analyse le format de titre Aurion sur 5 lignes
// Format : "Lieu\nInfoAdditionnelle\nMatière\nTypeCours\nProfesseur"
// Tous les champs sont optionnels et peuvent être vides
func parseTitle(title string) (location, additionalInfo, subject, courseType, professor string) {
	parts := strings.Split(title, "\n")

	if len(parts) > 0 {
		location = strings.TrimSpace(parts[0])
	}
	if len(parts) > 1 {
		additionalInfo = strings.TrimSpace(parts[1])
	}
	if len(parts) > 2 {
		subject = strings.TrimSpace(parts[2])
	}
	if len(parts) > 3 {
		courseType = strings.TrimSpace(parts[3])
	}
	if len(parts) > 4 {
		professor = strings.TrimSpace(parts[4])
	}

	return location, additionalInfo, subject, courseType, professor
}

// buildSummary construit le résumé de l'événement avec un préfixe emoji
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

// buildDescription construit la description de l'événement
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

// escapeText échappe les caractères spéciaux dans les valeurs texte iCal
// La RFC 5545 exige l'échappement de : backslash, virgule, point-virgule, saut de ligne
func escapeText(text string) string {
	// Order matters: escape backslash first
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, ",", "\\,")
	text = strings.ReplaceAll(text, ";", "\\;")
	text = strings.ReplaceAll(text, "\n", "\\n")
	return text
}

// foldLine plie une ligne à 75 octets selon la RFC 5545
// Les lignes de continuation commencent par un espace
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
		if currentPos > 0 {
			chunkSize = continuationLength
		}

		if remaining < chunkSize {
			chunkSize = remaining
		}

		chunk := string(bytes[currentPos : currentPos+chunkSize])
		chunks = append(chunks, chunk)
		currentPos += chunkSize
	}

	// Join with CRLF followed by space for continuation
	return strings.Join(chunks, "\r\n ")
}

// formatDateUTC formate un time.Time en horodatage iCal UTC (YYYYMMDDTHHMMSSZ)
func formatDateUTC(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// parseDate analyse une chaîne de date (timestamp ou format ISO) en time.Time
func parseDate(dateValue string) time.Time {
	// Match worker behavior: numeric strings are interpreted as milliseconds.
	numericDate := regexp.MustCompile(`^\d+$`)
	if numericDate.MatchString(dateValue) {
		ms, err := strconv.ParseInt(dateValue, 10, 64)
		if err == nil {
			return time.Unix(ms/1000, 0)
		}
	}

	// Try parsing as ISO 8601 format
	if t, err := time.Parse(time.RFC3339, dateValue); err == nil {
		return t
	}

	// Worker accepts JS Date parsing for bare datetime strings.
	if t, err := time.Parse("2006-01-02T15:04:05", dateValue); err == nil {
		return t
	}

	return time.Time{}
}

// normalizeSpaces normalise les espaces blancs dans le texte
func normalizeSpaces(text string) string {
	// Replace multiple spaces/tabs with single space
	fields := strings.Fields(text)
	return strings.Join(fields, " ")
}
