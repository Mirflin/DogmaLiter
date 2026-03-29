package models

import "time"

type CharacterInventory struct {
	ID            string `gorm:"type:varchar(36);primaryKey" json:"id"`
	CharacterID   string `gorm:"type:varchar(36);not null;index" json:"character_id"`
	ItemID        string `gorm:"type:varchar(36);not null;index" json:"item_id"`

	Quantity      int    `gorm:"default:1" json:"quantity"`
	Durability    *int   `gorm:"default:null" json:"durability"`
	MaxDurability *int   `gorm:"default:null" json:"max_durability"`
	Enchantment   int    `gorm:"default:0" json:"enchantment"`

	GridX         int    `gorm:"not null" json:"grid_x"`
	GridY         int    `gorm:"not null" json:"grid_y"`
	IsRotated     bool   `gorm:"default:false" json:"is_rotated"`

	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`

	Character Character `gorm:"foreignKey:CharacterID" json:"-"`
	Item      Item      `gorm:"foreignKey:ItemID" json:"item,omitempty"`
}