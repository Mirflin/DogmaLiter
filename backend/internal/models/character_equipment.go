package models

import "time"

type CharacterEquipment struct {
	CharacterID     string    `gorm:"type:varchar(36);primaryKey" json:"character_id"`
	Slot            string    `gorm:"type:varchar(30);primaryKey" json:"slot"`
	InventoryItemID string    `gorm:"type:varchar(36);not null" json:"inventory_item_id"`
	CreatedAt       time.Time `json:"created_at"`

	Character     Character          `gorm:"foreignKey:CharacterID" json:"-"`
	InventoryItem CharacterInventory `gorm:"foreignKey:InventoryItemID" json:"inventory_item,omitempty"`
}

const (
	SlotHead     = "head"
	SlotChest    = "chest"
	SlotGloves   = "gloves"
	SlotBelt     = "belt"
	SlotBoots    = "boots"
	SlotMainHand = "main_hand"
	SlotOffHand  = "off_hand"
	SlotRing1    = "ring_1"
	SlotRing2    = "ring_2"
	SlotAmulet   = "amulet"
)

var ValidSlots = []string{
	SlotHead, SlotChest, SlotGloves, SlotBelt, SlotBoots,
	SlotMainHand, SlotOffHand, SlotRing1, SlotRing2, SlotAmulet,
}
