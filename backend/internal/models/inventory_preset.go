package models

import "time"

// InventoryPreset is a backpack shape a GM draws once and hands out to
// characters. Applying it copies the shape onto the character, so editing or
// deleting the preset later never moves items already in someone's pack.
type InventoryPreset struct {
	ID     string `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID string `gorm:"type:varchar(36);not null;index" json:"game_id"`
	Name   string `gorm:"type:varchar(60);not null" json:"name"`
	Width  int    `gorm:"not null" json:"width"`
	Height int    `gorm:"not null" json:"height"`
	// Mask uses the same format as Character.InventoryMask and is never empty.
	Mask string `gorm:"type:varchar(400);not null" json:"mask"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
