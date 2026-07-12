package handler

// Recipient represents a structured recipient with name and email
type Recipient struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
