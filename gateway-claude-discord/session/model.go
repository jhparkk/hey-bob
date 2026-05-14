package session

import (
	"database/sql"
	"time"
)

type Session struct {
	ID              int64
	ChannelID       string
	ClaudeSessionID string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Model struct {
	db *sql.DB
}

func NewModel(db *sql.DB) *Model {
	return &Model{db: db}
}

func (m *Model) GetByChannelID(channelID string) (*Session, error) {
	row := m.db.QueryRow(
		`SELECT id, channel_id, claude_session_id, created_at, updated_at
		 FROM sessions WHERE channel_id = ?`,
		channelID,
	)

	s := &Session{}
	err := row.Scan(&s.ID, &s.ChannelID, &s.ClaudeSessionID, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (m *Model) Upsert(channelID, claudeSessionID string) error {
	_, err := m.db.Exec(`
		INSERT INTO sessions (channel_id, claude_session_id, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(channel_id) DO UPDATE SET
			claude_session_id = excluded.claude_session_id,
			updated_at = CURRENT_TIMESTAMP
	`, channelID, claudeSessionID)
	return err
}

func (m *Model) Delete(channelID string) error {
	_, err := m.db.Exec("DELETE FROM sessions WHERE channel_id = ?", channelID)
	return err
}
