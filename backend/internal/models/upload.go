package models

import "time"

type Upload struct {
	ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID       string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	FileType     string    `gorm:"type:enum('avatar','portrait','item_icon','game_cover','map','news_image');not null" json:"file_type"`
	OriginalName string    `gorm:"type:varchar(300);not null" json:"original_name"`
	StorageKey   string    `gorm:"type:varchar(500);not null" json:"storage_key"`
	MimeType     string    `gorm:"type:varchar(100);not null" json:"mime_type"`
	SizeBytes    int64     `gorm:"not null" json:"size_bytes"`
	Width        *int      `gorm:"default:null" json:"width"`
	Height       *int      `gorm:"default:null" json:"height"`
	CreatedAt    time.Time `json:"created_at"`
}
