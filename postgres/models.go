package postgres

import (
	"time"

	"github.com/GetStream/stream-backend-homework-assignment/api"
)

// A message represents a message in the database.
type message struct {
	ID          string     `bun:",pk,type:uuid,default:uuid_generate_v4()"`
	MessageText string     `bun:"message_text,notnull"`
	UserID      string     `bun:",notnull"`
	CreatedAt   time.Time  `bun:",nullzero,default:now()"`
	Reactions   []reaction `bun:"rel:has-many,join:id=message_id"`
}

type reaction struct {
	ID        string    `bun:",pk,type:uuid,default:uuid_generate_v4()"`
	MessageID string    `bun:",notnull"`
	UserID    string    `bun:",notnull"`
	Type      string    `bun:",notnull"`
	Score     int       `bun:",notnull,default:1"`
	CreatedAt time.Time `bun:",nullzero,default:now()"`
	Message   message   `bun:"rel:belongs-to,join:id=id"`
}

func (m message) APIMessage() api.Message {
	reactions := make([]api.Reaction, len(m.Reactions))
	for i, r := range m.Reactions {
		reactions[i] = r.APIReaction()
	}

	return api.Message{
		ID:            m.ID,
		Text:          m.MessageText,
		UserID:        m.UserID,
		CreatedAt:     m.CreatedAt,
		Reactions:     reactions,
		ReactionCount: len(m.Reactions),
	}
}

func (r reaction) APIReaction() api.Reaction {
	return api.Reaction{
		ID:        r.ID,
		MessageID: r.MessageID,
		UserID:    r.UserID,
		Type:      r.Type,
		Score:     r.Score,
		CreatedAt: r.CreatedAt,
	}
}
