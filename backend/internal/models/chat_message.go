package models

import "time"

type ChatMessage struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID      string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	UserID      string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	MessageType string    `gorm:"type:enum('text','dice_roll','item_link','system');default:'text'" json:"message_type"`
	Content     string    `gorm:"type:text;not null" json:"content"`
	Metadata    *string   `gorm:"type:json" json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`

	Game Game `gorm:"foreignKey:GameID" json:"game,omitempty"`
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}