package auth

import (
	"encoding/base64"
	"strings"
)

// Credentials représente l'email et le mot de passe décodés de l'authentification Basic.
type Credentials struct {
	Email    string // Email de l'utilisateur
	Password string // Mot de passe de l'utilisateur
}

// AuthResult représente le résultat de l'analyse d'un en-tête Basic Auth.
type AuthResult struct {
	Success     bool         // true si l'authentification est valide
	Credentials *Credentials // Identifiants décodés (nil si échec)
	FailReason  string       // "missing_header", "invalid_format", "decode_error"
}

// ParseBasicAuth analyse un en-tête HTTP Basic Auth et retourne un résultat structuré.
// Gère les cas d'erreur courants avec des messages descriptifs.
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
