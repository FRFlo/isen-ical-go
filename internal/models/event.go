// Package models fournit les structures de données pour l'application.
// Ce package contient tous les modèles de domaine utilisés dans l'application.
package models

// AurionEvent représente un événement de calendrier récupéré du système Aurion.
// Il suit le format standard des événements calendaires avec les dates de début/fin et métadonnées.
type AurionEvent struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Start     string `json:"start"`
	End       string `json:"end"`
	AllDay    bool   `json:"allDay"`
	Editable  bool   `json:"editable"`
	ClassName string `json:"className"`
}
