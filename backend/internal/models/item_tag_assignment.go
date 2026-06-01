package models

import "time"

type ItemTagAssignment struct {
	ItemID        string    `gorm:"type:varchar(36);primaryKey" json:"item_id"`
	GameItemTagID string    `gorm:"type:varchar(36);primaryKey;index" json:"game_item_tag_id"`
	CreatedAt     time.Time `json:"created_at"`

	Item        Item        `gorm:"foreignKey:ItemID" json:"-"`
	GameItemTag GameItemTag `gorm:"foreignKey:GameItemTagID" json:"-"`
}
