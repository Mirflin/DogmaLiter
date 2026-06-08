package models

import "time"

type TradeOffer struct {
	ID                string     `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID            string     `gorm:"type:varchar(36);not null;index" json:"game_id"`
	FromUserID        string     `gorm:"type:varchar(36);not null;index" json:"from_user_id"`
	FromCharacterID   string     `gorm:"type:varchar(36);not null" json:"from_character_id"`
	FromUsername      string     `gorm:"type:varchar(50)" json:"from_username"`
	FromCharacterName string     `gorm:"type:varchar(100)" json:"from_character_name"`
	ToUserID          string     `gorm:"type:varchar(36);not null;index" json:"to_user_id"`
	ToCharacterID     string     `gorm:"type:varchar(36);not null" json:"to_character_id"`
	ToUsername        string     `gorm:"type:varchar(50)" json:"to_username"`
	ToCharacterName   string     `gorm:"type:varchar(100)" json:"to_character_name"`
	Status            string     `gorm:"type:varchar(20);default:'pending';index" json:"status"`
	CreatedAt         time.Time  `json:"created_at"`
	RespondedAt       *time.Time `json:"responded_at"`

	Items []TradeOfferItem `gorm:"foreignKey:TradeOfferID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
}

type TradeOfferItem struct {
	ID            string `gorm:"type:varchar(36);primaryKey" json:"id"`
	TradeOfferID  string `gorm:"type:varchar(36);not null;index" json:"trade_offer_id"`
	ItemID        string `gorm:"type:varchar(36);not null;index" json:"item_id"`
	Quantity      int    `gorm:"default:1" json:"quantity"`
	Durability    *int   `json:"durability"`
	MaxDurability *int   `json:"max_durability"`
	Enchantment   int    `gorm:"default:0" json:"enchantment"`

	Item Item `gorm:"foreignKey:ItemID;constraint:OnDelete:CASCADE" json:"item,omitempty"`
}
