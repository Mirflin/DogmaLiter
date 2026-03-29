package models

import "time"

type Game struct {
	ID                string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	OwnerID           string    `gorm:"type:varchar(36);not null;index" json:"owner_id"`
	Title             string    `gorm:"type:varchar(200);not null" json:"title"`
	Description       string    `gorm:"type:text" json:"description"`
	System            string    `gorm:"type:varchar(50);default:'dnd5e'" json:"system"`
	InviteCode        string    `gorm:"type:varchar(20);uniqueIndex" json:"invite_code"`
	MaxPlayers        int       `gorm:"default:6" json:"max_players"`
	IsActive          bool      `gorm:"default:true" json:"is_active"`
	CoverImageID      *string   `gorm:"type:varchar(36)" json:"cover_image_id"`
	ShowStandardAttrs bool      `gorm:"default:true" json:"show_standard_attrs"`
	EnableChat        bool      `gorm:"default:true" json:"enable_chat"`
	EnableItemTrading bool      `gorm:"default:true" json:"enable_item_trading"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`

	Owner      User          `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	CoverImage *Upload       `gorm:"foreignKey:CoverImageID" json:"cover_image,omitempty"`
	Members    []GameMember  `gorm:"foreignKey:GameID" json:"members,omitempty"`
	Characters []Character   `gorm:"foreignKey:GameID" json:"characters,omitempty"`
	Items      []Item        `gorm:"foreignKey:GameID" json:"items,omitempty"`
	Maps       []GameMap     `gorm:"foreignKey:GameID" json:"maps,omitempty"`
}