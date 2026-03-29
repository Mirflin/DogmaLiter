package models

import "time"

type User struct {
	ID           string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Username     string    `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Email        string    `gorm:"type:varchar(200);uniqueIndex;not null" json:"email"`
	PasswordHash *string   `gorm:"type:varchar(200)" json:"-"`
	GoogleID     *string   `gorm:"type:varchar(100);uniqueIndex" json:"-"`
	Role         string    `gorm:"type:enum('user','admin');default:'user'" json:"role"`
	AvatarID     *string   `gorm:"type:varchar(36)" json:"avatar_id"`
	PlanID       string    `gorm:"type:varchar(30);default:'free'" json:"plan_id"`
	IsVerified   bool      `gorm:"default:false" json:"is_verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Plan       Plan        `gorm:"foreignKey:PlanID" json:"plan,omitempty"`
	OwnedGames []Game      `gorm:"foreignKey:OwnerID" json:"owned_games,omitempty"`
	Characters []Character `gorm:"foreignKey:UserID" json:"characters,omitempty"`
}
