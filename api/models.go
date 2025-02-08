package api

import "time"

// A Message represents a persisted message.
type Message struct {
	ID            string     `json:"id"`
	Text          string     `json:"text"`
	UserID        string     `json:"user_id"`
	CreatedAt     time.Time  `json:"created_at"`
	Reactions     []Reaction `json:"reactions"`
	ReactionCount int        `json:"reaction_count"`
}

// A Reaction represents a reaction to a message such as a like.
type Reaction struct {
	ID        string    `json:"id"`
	MessageID string    `json:"message_id"`
	Type      string    `json:"type"`
	Score     int       `json:"score"`
	UserID    string    `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
