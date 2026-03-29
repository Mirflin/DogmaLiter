package models

import "time"

type Item struct {
	ID          string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID      string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	CreatedByID string    `gorm:"type:varchar(36);not null" json:"created_by_id"`

	Name        string  `gorm:"type:varchar(200);not null" json:"name"`
	Description string  `gorm:"type:text" json:"description"`
	ImageID     *string `gorm:"type:varchar(36)" json:"image_id"`

	Rarity      string  `gorm:"type:enum('common','uncommon','rare','epic','legendary','artifact');default:'common'" json:"rarity"`
	GridWidth   int     `gorm:"default:1" json:"grid_width"`
	GridHeight  int     `gorm:"default:1" json:"grid_height"`
	IsEquippable bool   `gorm:"default:false" json:"is_equippable"`
	EquipSlot   *string `gorm:"type:varchar(30)" json:"equip_slot"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Game               Game                    `gorm:"foreignKey:GameID" json:"game,omitempty"`
	CreatedBy          User                    `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	Image              *Upload                 `gorm:"foreignKey:ImageID" json:"image,omitempty"`
	Types              []ItemType              `gorm:"foreignKey:ItemID" json:"types,omitempty"`
	RequiredAttributes []ItemRequiredAttribute `gorm:"foreignKey:ItemID" json:"required_attributes,omitempty"`
	AttributeModifiers []ItemAttributeModifier `gorm:"foreignKey:ItemID" json:"attribute_modifiers,omitempty"`
}