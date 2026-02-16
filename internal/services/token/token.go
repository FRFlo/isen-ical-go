// Package token fournit le chiffrement AES-GCM et la gestion des tokens
package token

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/FRFlo/isen-ical-go/internal/models"
	"github.com/FRFlo/isen-ical-go/internal/storage"
	"github.com/google/uuid"
)

const (
	KeySize          = 32
	MaxTokensPerUser = 3
)

// Service fournit le chiffrement et la gestion des tokens
type Service struct {
	valkey *storage.ValkeyClient
}

// NewService crée un nouveau service de token
func NewService(valkey *storage.ValkeyClient) *Service {
	return &Service{valkey: valkey}
}

// Encrypt chiffre un texte avec AES-GCM et la clé fournie
// Retourne les données chiffrées et le IV utilisé
func Encrypt(plaintext []byte, key []byte) (encrypted []byte, iv []byte, err error) {
	if len(key) != KeySize {
		return nil, nil, fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	iv = make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	encrypted = gcm.Seal(nil, iv, plaintext, nil)
	return encrypted, iv, nil
}

// Decrypt decrypts data using AES-GCM with the provided key and IV
func Decrypt(encrypted []byte, iv []byte, key []byte) (plaintext []byte, err error) {
	if len(key) != KeySize {
		return nil, fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(iv) != gcm.NonceSize() {
		return nil, fmt.Errorf("invalid IV size: expected %d bytes, got %d", gcm.NonceSize(), len(iv))
	}

	plaintext, err = gcm.Open(nil, iv, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// GenerateToken creates a new token for the given credentials
// Returns the token ID, the encryption key (base64url encoded), and the URL-safe token string
func (s *Service) GenerateToken(email, password string) (tokenID string, key string, urlToken string, err error) {
	tokenID = uuid.New().String()

	keyBytes := make([]byte, KeySize)
	if _, err := rand.Read(keyBytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate key: %w", err)
	}

	key = base64.RawURLEncoding.EncodeToString(keyBytes)

	creds := models.Credentials{
		Email:    email,
		Password: password,
	}

	credsJSON, err := json.Marshal(creds)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal credentials: %w", err)
	}

	encrypted, iv, err := Encrypt(credsJSON, keyBytes)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to encrypt credentials: %w", err)
	}

	storedToken := models.StoredToken{
		Encrypted: encrypted,
		IV:        iv,
		CreatedAt: time.Now(),
		Email:     email,
	}

	tokenJSON, err := json.Marshal(storedToken)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to marshal token: %w", err)
	}

	tokenKey := storage.TokenKey(tokenID)
	if err := s.valkey.Set(tokenKey, tokenJSON, 0); err != nil {
		return "", "", "", fmt.Errorf("failed to store token: %w", err)
	}

	if err := s.addTokenToUserList(email, tokenID); err != nil {
		return "", "", "", fmt.Errorf("failed to add token to user list: %w", err)
	}

	urlToken = fmt.Sprintf("%s:%s", tokenID, key)

	return tokenID, key, urlToken, nil
}

func (s *Service) addTokenToUserList(email, tokenID string) error {
	userTokensKey := storage.UserTokensKey(email)

	existingTokensJSON, err := s.valkey.Get(userTokensKey)
	var userTokens []models.UserTokenEntry

	if err == nil {
		if err := json.Unmarshal([]byte(existingTokensJSON), &userTokens); err != nil {
			return fmt.Errorf("failed to unmarshal user tokens: %w", err)
		}
	}

	newEntry := models.UserTokenEntry{
		Token:     tokenID,
		CreatedAt: time.Now(),
	}
	userTokens = append(userTokens, newEntry)

	if len(userTokens) > MaxTokensPerUser {
		tokensToRemove := userTokens[:len(userTokens)-MaxTokensPerUser]
		userTokens = userTokens[len(userTokens)-MaxTokensPerUser:]

		for _, entry := range tokensToRemove {
			tokenKey := storage.TokenKey(entry.Token)
			err := s.valkey.Delete(tokenKey)
			if err != nil {
				fmt.Printf("Warning: failed to delete old token %s: %v\n", entry.Token, err)
				continue
			}
		}
	}

	updatedTokensJSON, err := json.Marshal(userTokens)
	if err != nil {
		return fmt.Errorf("failed to marshal user tokens: %w", err)
	}

	if err := s.valkey.Set(userTokensKey, updatedTokensJSON, 0); err != nil {
		return fmt.Errorf("failed to store user tokens: %w", err)
	}

	return nil
}

// GetTokenData retrieves the stored token data from Valkey
func (s *Service) GetTokenData(tokenID string) (*models.StoredToken, error) {
	tokenKey := storage.TokenKey(tokenID)

	tokenJSON, err := s.valkey.Get(tokenKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}

	var token models.StoredToken
	if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// DecryptCredentials decrypts the stored token credentials using the provided key
func DecryptCredentials(token *models.StoredToken, key []byte) (*models.Credentials, error) {
	plaintext, err := Decrypt(token.Encrypted, token.IV, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	var creds models.Credentials
	if err := json.Unmarshal(plaintext, &creds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal credentials: %w", err)
	}

	return &creds, nil
}

// ParseURLToken parses a URL-safe token string into token ID and key
func ParseURLToken(urlToken string) (tokenID string, key []byte, err error) {
	for i, c := range urlToken {
		if c == ':' {
			tokenID = urlToken[:i]
			keyStr := urlToken[i+1:]

			if tokenID == "" || keyStr == "" {
				return "", nil, fmt.Errorf("invalid token format: empty components")
			}

			key, err = base64.RawURLEncoding.DecodeString(keyStr)
			if err != nil {
				return "", nil, fmt.Errorf("failed to decode key: %w", err)
			}

			if len(key) != KeySize {
				return "", nil, fmt.Errorf("invalid key size: expected %d bytes, got %d", KeySize, len(key))
			}

			return tokenID, key, nil
		}
	}

	return "", nil, fmt.Errorf("invalid token format: missing separator")
}
