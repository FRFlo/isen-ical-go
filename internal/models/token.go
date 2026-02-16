// Package models fournit les structures de données pour l'application.
package models

import "time"

// StoredToken représente les identifiants chiffrés stockés dans Valkey.
type StoredToken struct {
	Encrypted []byte    `json:"encrypted"`
	IV        []byte    `json:"iv"`
	CreatedAt time.Time `json:"createdAt"`
	Email     string    `json:"email"`
}

// Credentials représente les identifiants utilisateur déchiffrés.
type Credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserTokenEntry représente une entrée de token unique dans la liste des tokens d'un utilisateur.
type UserTokenEntry struct {
	Token     string    `json:"token"`
	CreatedAt time.Time `json:"createdAt"`
}
