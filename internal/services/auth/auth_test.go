package auth

import (
	"encoding/base64"
	"testing"
)

func TestParseBasicAuth(t *testing.T) {
	tests := []struct {
		name           string
		header         string
		wantSuccess    bool
		wantFailReason string
		wantEmail      string
		wantPassword   string
	}{
		{
			name:           "empty header returns missing_header",
			header:         "",
			wantSuccess:    false,
			wantFailReason: "missing_header",
		},
		{
			name:           "missing Basic prefix returns invalid_format",
			header:         "Bearer token123",
			wantSuccess:    false,
			wantFailReason: "invalid_format",
		},
		{
			name:           "lowercase basic prefix returns invalid_format",
			header:         "basic dXNlcjpwYXNz",
			wantSuccess:    false,
			wantFailReason: "invalid_format",
		},
		{
			name:           "invalid base64 returns decode_error",
			header:         "Basic !!!invalid!!!",
			wantSuccess:    false,
			wantFailReason: "decode_error",
		},
		{
			name:           "no colon in decoded string returns invalid_format",
			header:         "Basic " + base64.StdEncoding.EncodeToString([]byte("justemail")),
			wantSuccess:    false,
			wantFailReason: "invalid_format",
		},
		{
			name:         "valid credentials with simple password",
			header:       "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:password123")),
			wantSuccess:  true,
			wantEmail:    "user@example.com",
			wantPassword: "password123",
		},
		{
			name:         "valid credentials with password containing colons",
			header:       "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:pass:word:123")),
			wantSuccess:  true,
			wantEmail:    "user@example.com",
			wantPassword: "pass:word:123",
		},
		{
			name:         "valid credentials with empty password",
			header:       "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:")),
			wantSuccess:  true,
			wantEmail:    "user@example.com",
			wantPassword: "",
		},
		{
			name:         "valid credentials with special characters in password",
			header:       "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:p@$$w0rd!")),
			wantSuccess:  true,
			wantEmail:    "user@example.com",
			wantPassword: "p@$$w0rd!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBasicAuth(tt.header)

			if result.Success != tt.wantSuccess {
				t.Errorf("ParseBasicAuth() Success = %v, want %v", result.Success, tt.wantSuccess)
			}

			if !tt.wantSuccess {
				if result.FailReason != tt.wantFailReason {
					t.Errorf("ParseBasicAuth() FailReason = %v, want %v", result.FailReason, tt.wantFailReason)
				}
				if result.Credentials != nil {
					t.Errorf("ParseBasicAuth() Credentials should be nil on failure, got %v", result.Credentials)
				}
			} else {
				if result.Credentials == nil {
					t.Errorf("ParseBasicAuth() Credentials should not be nil on success")
					return
				}
				if result.Credentials.Email != tt.wantEmail {
					t.Errorf("ParseBasicAuth() Email = %v, want %v", result.Credentials.Email, tt.wantEmail)
				}
				if result.Credentials.Password != tt.wantPassword {
					t.Errorf("ParseBasicAuth() Password = %v, want %v", result.Credentials.Password, tt.wantPassword)
				}
			}
		})
	}
}
