package models

import "time"

type ActivityLog struct {
	ID            string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID        string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	UserID        string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	CharacterName string    `gorm:"type:varchar(100)" json:"character_name"`
	Action        string    `gorm:"type:varchar(60);not null" json:"action"`
	Details       string    `gorm:"type:varchar(300)" json:"details"`
	CreatedAt     time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
