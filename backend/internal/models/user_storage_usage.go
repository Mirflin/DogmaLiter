package models

import "time"

type UserStorageUsage struct {
	UserID     string    `gorm:"type:varchar(36);primaryKey" json:"user_id"`
	UsedBytes  int64     `gorm:"default:0" json:"used_bytes"`
	FilesCount int       `gorm:"default:0" json:"files_count"`
	UpdatedAt  time.Time `json:"updated_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}