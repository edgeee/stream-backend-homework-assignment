package redis

import (
	"time"

	"github.com/GetStream/stream-backend-homework-assignment/api"
)

// A message represents a message in the database.
type message struct {
	ID        string    `redis:"id"`
	Text      string    `redis:"text"`
	UserID    string    `redis:"user_id"`
	CreatedAt time.Time `redis:"created_at"`
	Reactions []reaction
}

// reaction represents a reaction to a message, stored in the database.
type reaction struct {
	ID        string    `redis:"id"`
	MessageID string    `redis:"message_id"`
	UserID    string    `redis:"user_id"`
	Type      string    `redis:"type"`
	Score     int       `redis:"score"`
	CreatedAt time.Time `redis:"created_at"`
}

func (m message) APIMessage() api.Message {
	rcs := make([]api.Reaction, len(m.Reactions))
	for i, r := range m.Reactions {
		rcs[i] = r.APIReaction()
	}

	apiMsg := api.Message{
		ID:            m.ID,
		Text:          m.Text,
		UserID:        m.UserID,
		CreatedAt:     m.CreatedAt,
		Reactions:     rcs,
		ReactionCount: len(m.Reactions),
	}
	return apiMsg
}

func (r reaction) APIReaction() api.Reaction {
	return api.Reaction{
		ID:        r.ID,
		MessageID: r.MessageID,
		UserID:    r.UserID,
		Type:      r.Type,
		Score:     r.Score,
	}
}
