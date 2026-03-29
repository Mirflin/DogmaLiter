package models

import "time"

type Character struct {
	ID              string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID          string    `gorm:"type:varchar(36);not null;index" json:"game_id"`
	UserID          string    `gorm:"type:varchar(36);not null;index" json:"user_id"`
	CreatedByID     string    `gorm:"type:varchar(36);not null" json:"created_by_id"`

	Name            string    `gorm:"type:varchar(100);not null" json:"name"`
	Backstory       string    `gorm:"type:text" json:"backstory"`
	PortraitID      *string   `gorm:"type:varchar(36)" json:"portrait_id"`

	BaseStrength     int `gorm:"default:10" json:"base_strength"`
	BaseDexterity    int `gorm:"default:10" json:"base_dexterity"`
	BaseConstitution int `gorm:"default:10" json:"base_constitution"`
	BaseIntelligence int `gorm:"default:10" json:"base_intelligence"`
	BaseWisdom       int `gorm:"default:10" json:"base_wisdom"`
	BaseCharisma     int `gorm:"default:10" json:"base_charisma"`

	InventoryWidth  int `gorm:"default:10" json:"inventory_width"`
	InventoryHeight int `gorm:"default:6" json:"inventory_height"`

	CurrencyGold   int `gorm:"default:0" json:"currency_gold"`
	CurrencySilver int `gorm:"default:0" json:"currency_silver"`
	CurrencyCopper int `gorm:"default:0" json:"currency_copper"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Game             Game                     `gorm:"foreignKey:GameID" json:"game,omitempty"`
	User             User                     `gorm:"foreignKey:UserID" json:"user,omitempty"`
	CreatedBy        User                     `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	Portrait         *Upload                  `gorm:"foreignKey:PortraitID" json:"portrait,omitempty"`
	CustomAttributes []CharacterCustomAttribute `gorm:"foreignKey:CharacterID" json:"custom_attributes,omitempty"`
	Inventory        []CharacterInventory     `gorm:"foreignKey:CharacterID" json:"inventory,omitempty"`
	Equipment        []CharacterEquipment     `gorm:"foreignKey:CharacterID" json:"equipment,omitempty"`
}