// Package models provides data structures for the application
package models

import "time"

// StoredToken represents encrypted credentials stored in Valkey
type StoredToken struct {
	Encrypted []byte    `json:"encrypted"`
	IV        []byte    `json:"iv"`
	CreatedAt time.Time `json:"createdAt"`
	Email     string    `json:"email"`
}

// Credentials represents decrypted user credentials
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserTokenEntry represents a single token entry in a user's token list
type UserTokenEntry struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}
