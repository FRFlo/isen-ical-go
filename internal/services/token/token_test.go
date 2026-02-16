package token

import (
	"encoding/base64"
	"testing"
)

func TestEncryptDecryptRoundtrip(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("test credentials data")

	encrypted, iv, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if len(iv) == 0 {
		t.Error("IV should not be empty")
	}

	decrypted, err := Decrypt(encrypted, iv, key)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted data mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptInvalidKeySize(t *testing.T) {
	key := []byte("short")
	plaintext := []byte("test")

	_, _, err := Encrypt(plaintext, key)
	if err == nil {
		t.Error("Encrypt should fail with invalid key size")
	}
}

func TestDecryptInvalidKeySize(t *testing.T) {
	key := []byte("short")
	encrypted := []byte("encrypted")
	iv := make([]byte, 12)

	_, err := Decrypt(encrypted, iv, key)
	if err == nil {
		t.Error("Decrypt should fail with invalid key size")
	}
}

func TestDecryptTamperedData(t *testing.T) {
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("test credentials data")

	encrypted, iv, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	encrypted[0] ^= 0xFF

	_, err = Decrypt(encrypted, iv, key)
	if err == nil {
		t.Error("Decrypt should fail with tampered data")
	}
}

func TestParseURLToken(t *testing.T) {
	tests := []struct {
		name     string
		urlToken string
		wantErr  bool
		errMsg   string
		wantID   string
		wantKey  []byte
	}{
		{
			name:     "valid token",
			urlToken: "550e8400-e29b-41d4-a716-446655440000:dGhpc2lzYV90ZXN0a2V5dGhhdGlzMzJieXRlc2xvbmc",
			wantErr:  false,
			wantID:   "550e8400-e29b-41d4-a716-446655440000",
			wantKey:  []byte("thisisa_testkeythatis32byteslong"),
		},
		{
			name:     "missing separator",
			urlToken: "invalidtoken",
			wantErr:  true,
			errMsg:   "missing separator",
		},
		{
			name:     "empty token ID",
			urlToken: ":keydata",
			wantErr:  true,
			errMsg:   "empty components",
		},
		{
			name:     "empty key",
			urlToken: "tokenid:",
			wantErr:  true,
			errMsg:   "empty components",
		},
		{
			name:     "invalid base64",
			urlToken: "tokenid:not-valid-base64!!!",
			wantErr:  true,
			errMsg:   "failed to decode key",
		},
		{
			name:     "wrong key size",
			urlToken: "tokenid:Zm9vYmFy",
			wantErr:  true,
			errMsg:   "invalid key size",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenID, key, err := ParseURLToken(tt.urlToken)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseURLToken() error = nil, want error containing %q", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("ParseURLToken() error = %v, want error containing %q", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseURLToken() unexpected error = %v", err)
				return
			}

			if tokenID != tt.wantID {
				t.Errorf("ParseURLToken() tokenID = %v, want %v", tokenID, tt.wantID)
			}

			if !bytesEqual(key, tt.wantKey) {
				t.Errorf("ParseURLToken() key = %v, want %v", key, tt.wantKey)
			}
		})
	}
}

func TestParseURLTokenWithGeneratedKey(t *testing.T) {
	tokenID := "test-token-id"
	key := make([]byte, KeySize)
	for i := range key {
		key[i] = byte(i + 1)
	}

	encodedKey := base64.RawURLEncoding.EncodeToString(key)
	urlToken := tokenID + ":" + encodedKey

	parsedID, parsedKey, err := ParseURLToken(urlToken)
	if err != nil {
		t.Fatalf("ParseURLToken failed: %v", err)
	}

	if parsedID != tokenID {
		t.Errorf("tokenID mismatch: got %q, want %q", parsedID, tokenID)
	}

	if len(parsedKey) != KeySize {
		t.Errorf("key size mismatch: got %d, want %d", len(parsedKey), KeySize)
	}

	for i := range key {
		if parsedKey[i] != key[i] {
			t.Errorf("key mismatch at index %d: got %d, want %d", i, parsedKey[i], key[i])
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func bytesEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
