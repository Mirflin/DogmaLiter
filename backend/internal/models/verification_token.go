package models

import "time"

type VerificationToken struct {
	ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	UserID    string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	Token     string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"-"`
	Type      string    `gorm:"type:enum('email_verify','password_reset');not null" json:"type"`
	ExpiresAt time.Time `gorm:"not null" json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`

	User User `gorm:"foreignKey:UserID" json:"-"`
}
