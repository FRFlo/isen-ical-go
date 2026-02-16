package auth

import (
	"encoding/base64"
	"strings"
)

// Credentials represents the decoded email and password from Basic Auth
type Credentials struct {
	Email    string
	Password string
}

// AuthResult represents the result of parsing a Basic Auth header
type AuthResult struct {
	Success     bool
	Credentials *Credentials
	FailReason  string // "missing_header", "invalid_format", "decode_error"
}

// ParseBasicAuth parses an HTTP Basic Auth header and returns structured result
func ParseBasicAuth(header string) AuthResult {
	// Check for empty header
	if header == "" {
		return AuthResult{
			Success:    false,
			FailReason: "missing_header",
		}
	}

	// Check for "Basic " prefix (case-sensitive)
	const basicPrefix = "Basic "
	if !strings.HasPrefix(header, basicPrefix) {
		return AuthResult{
			Success:    false,
			FailReason: "invalid_format",
		}
	}

	// Extract the Base64 encoded credentials
	encodedCreds := header[len(basicPrefix):]

	// Base64 decode
	decodedBytes, err := base64.StdEncoding.DecodeString(encodedCreds)
	if err != nil {
		return AuthResult{
			Success:    false,
			FailReason: "decode_error",
		}
	}

	// Split on FIRST colon only (password may contain colons)
	decodedStr := string(decodedBytes)
	colonIdx := strings.Index(decodedStr, ":")
	if colonIdx == -1 {
		// No colon found - invalid format
		return AuthResult{
			Success:    false,
			FailReason: "invalid_format",
		}
	}

	email := decodedStr[:colonIdx]
	password := decodedStr[colonIdx+1:]

	return AuthResult{
		Success: true,
		Credentials: &Credentials{
			Email:    email,
			Password: password,
		},
	}
}
