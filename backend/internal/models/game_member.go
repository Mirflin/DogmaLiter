package models

import "time"

type GameMember struct {
	GameID   string    `gorm:"type:varchar(36);primaryKey" json:"game_id"`
	UserID   string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	Role     string    `gorm:"type:enum('gm','assistant_gm','player');default:'player'" json:"role"`
	JoinedAt time.Time `json:"joined_at"`

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Game Game `gorm:"foreignKey:GameID" json:"game,omitempty"`
}