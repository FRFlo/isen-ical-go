package models

// AurionEvent represents a calendar event from Aurion
type AurionEvent struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Start     string `json:"start"`
	End       string `json:"end"`
	AllDay    bool   `json:"allDay"`
	Editable  bool   `json:"editable"`
	ClassName string `json:"className"`
}
