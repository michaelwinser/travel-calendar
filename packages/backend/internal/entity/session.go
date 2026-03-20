package entity

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Session represents an authenticated user session.
type Session struct {
	ID        string    `db:"id" firestore:"-"`
	UserID    string    `db:"user_id" firestore:"userId"`
	Email     string    `db:"email" firestore:"email"`
	ExpiresAt time.Time `db:"expires_at" firestore:"expiresAt"`
	CreatedAt time.Time `db:"created_at" firestore:"createdAt"`
}

// IsExpired returns true if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// NewSession creates a new session with a random ID.
func NewSession(userID, email string, ttl time.Duration) Session {
	return Session{
		ID:        generateSessionID(),
		UserID:    userID,
		Email:     email,
		ExpiresAt: time.Now().Add(ttl),
		CreatedAt: time.Now(),
	}
}

// generateSessionID creates a cryptographically random session ID.
func generateSessionID() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("failed to generate session ID: " + err.Error())
	}
	return hex.EncodeToString(b)
}
