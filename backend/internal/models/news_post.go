package models

import "time"

type NewsPost struct {
	ID          string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	AuthorID    string     `gorm:"type:varchar(36);not null;index" json:"author_id"`
	Title       string     `gorm:"type:varchar(300);not null" json:"title"`
	Content     string     `gorm:"type:text;not null" json:"content"`
	ImageID     *string    `gorm:"type:varchar(36)" json:"image_id"`
	IsPublished bool       `gorm:"default:false" json:"is_published"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	Author User    `gorm:"foreignKey:AuthorID" json:"author,omitempty"`
	Image  *Upload `gorm:"foreignKey:ImageID" json:"image,omitempty"`
}