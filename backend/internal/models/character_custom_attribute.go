package models

import "time"

type CharacterCustomAttribute struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	CharacterID string    `gorm:"type:varchar(36);not null;index" json:"character_id"`
	Name        string    `gorm:"type:varchar(100);not null" json:"name"`
	Value       int       `gorm:"default:0" json:"value"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`

	Character Character `gorm:"foreignKey:CharacterID" json:"-"`
}