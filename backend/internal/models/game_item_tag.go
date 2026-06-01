package models

import "time"

type GameItemTag struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID      string    `gorm:"type:varchar(36);not null;uniqueIndex:idx_game_item_tags_game_name,priority:1" json:"game_id"`
	CreatedByID string    `gorm:"type:varchar(36);not null" json:"created_by_id"`
	Name        string    `gorm:"type:varchar(60);not null;uniqueIndex:idx_game_item_tags_game_name,priority:2" json:"name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`

	Game      Game   `gorm:"foreignKey:GameID" json:"-"`
	CreatedBy User   `gorm:"foreignKey:CreatedByID" json:"-"`
	Items     []Item `gorm:"many2many:item_tag_assignments;" json:"-"`
}
