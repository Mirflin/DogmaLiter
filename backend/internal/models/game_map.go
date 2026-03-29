package models

import "time"

type GameMap struct {
	ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID       string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	UploadedByID string    `gorm:"type:varchar(36);not null" json:"uploaded_by_id"`
	ImageID      string    `gorm:"type:varchar(36);not null" json:"image_id"`
	Name         string    `gorm:"type:varchar(200);not null" json:"name"`
	GridSize     int       `gorm:"default:50" json:"grid_size"`
	IsActive     bool      `gorm:"default:false" json:"is_active"`
	SortOrder    int       `gorm:"default:0" json:"sort_order"`
	CreatedAt    time.Time `json:"created_at"`

	Game       Game   `gorm:"foreignKey:GameID" json:"game,omitempty"`
	UploadedBy User   `gorm:"foreignKey:UploadedByID" json:"uploaded_by,omitempty"`
	Image      Upload `gorm:"foreignKey:ImageID" json:"image,omitempty"`
}