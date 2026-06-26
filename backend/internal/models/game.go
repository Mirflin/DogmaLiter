package models

import "time"

type Game struct {
	ID                  string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	OwnerID             string     `gorm:"type:varchar(36);not null;index" json:"owner_id"`
	Title               string     `gorm:"type:varchar(200);not null" json:"title"`
	Description         string     `gorm:"type:text" json:"description"`
	System              string     `gorm:"type:varchar(50);default:'dnd5e'" json:"system"`
	InviteCode          string     `gorm:"type:varchar(20);uniqueIndex" json:"invite_code"`
	InviteCodeExpiresAt *time.Time `json:"invite_code_expires_at"`
	MaxPlayers          int        `gorm:"default:6" json:"max_players"`
	CoverImageID        *string    `gorm:"type:varchar(36)" json:"cover_image_id"`
	ShowStandardAttrs   bool       `gorm:"default:true" json:"show_standard_attrs"`
	// EnabledStandardAttrs is the source of truth for which of the six base
	// attributes are active in a game (CSV of canonical keys). ShowStandardAttrs
	// is kept in sync as (len > 0) for backward compatibility.
	EnabledStandardAttrs string `gorm:"type:varchar(120);default:'strength,dexterity,constitution,intelligence,wisdom,charisma'" json:"-"`
	EnableChat           bool   `gorm:"default:true" json:"enable_chat"`
	EnableItemTrading    bool   `gorm:"default:true" json:"enable_item_trading"`
	EnableHealth         bool   `gorm:"default:false" json:"enable_health"`
	EnableArmorClass     bool   `gorm:"default:false" json:"enable_armor_class"`
	// CharacterSlotsPerPlayer is how many characters each member may own. The
	// game-wide cap scales as this value times the number of members.
	CharacterSlotsPerPlayer int       `gorm:"default:5" json:"character_slots_per_player"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`

	Owner      User         `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	CoverImage *Upload      `gorm:"foreignKey:CoverImageID" json:"cover_image,omitempty"`
	Members    []GameMember `gorm:"foreignKey:GameID" json:"members,omitempty"`
	Characters []Character  `gorm:"foreignKey:GameID" json:"characters,omitempty"`
	Items      []Item       `gorm:"foreignKey:GameID" json:"items,omitempty"`
	Maps       []GameMap    `gorm:"foreignKey:GameID" json:"maps,omitempty"`
}
