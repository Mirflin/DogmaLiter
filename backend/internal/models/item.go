package models

import "time"

const (
	ItemRarityCommon     = "common"
	ItemRarityUncommon   = "uncommon"
	ItemRarityRare       = "rare"
	ItemRarityEpic       = "epic"
	ItemRarityMasterwork = "masterwork"
	ItemRarityLegendary  = "legendary"
	ItemRarityUnique     = "unique"

	ItemCategoryLoot       = "loot"
	ItemCategoryConsumable = "consumable"
	ItemCategoryEquipment  = "equipment"
	ItemCategoryOther      = "other"

	ItemEquipSlotHead     = "head"
	ItemEquipSlotChest    = "chest"
	ItemEquipSlotGloves   = "gloves"
	ItemEquipSlotBelt     = "belt"
	ItemEquipSlotBoots    = "boots"
	ItemEquipSlotMainHand = "main_hand"
	ItemEquipSlotOffHand  = "off_hand"
	ItemEquipSlotRing     = "ring"
	ItemEquipSlotAmulet   = "amulet"
)

var ValidItemRarities = []string{
	ItemRarityCommon,
	ItemRarityUncommon,
	ItemRarityRare,
	ItemRarityEpic,
	ItemRarityMasterwork,
	ItemRarityLegendary,
	ItemRarityUnique,
}

var ValidItemCategories = []string{
	ItemCategoryLoot,
	ItemCategoryConsumable,
	ItemCategoryEquipment,
	ItemCategoryOther,
}

var ValidItemEquipSlots = []string{
	ItemEquipSlotHead,
	ItemEquipSlotChest,
	ItemEquipSlotGloves,
	ItemEquipSlotBelt,
	ItemEquipSlotBoots,
	ItemEquipSlotMainHand,
	ItemEquipSlotOffHand,
	ItemEquipSlotRing,
	ItemEquipSlotAmulet,
}

type Item struct {
	ID          string `gorm:"type:varchar(36);primaryKey" json:"id"`
	GameID      string `gorm:"type:varchar(36);not null;index" json:"game_id"`
	CreatedByID string `gorm:"type:varchar(36);not null" json:"created_by_id"`

	Name        string  `gorm:"type:varchar(200);not null" json:"name"`
	Description string  `gorm:"type:text" json:"description"`
	ImageID     *string `gorm:"type:varchar(36)" json:"image_id"`

	Rarity     string  `gorm:"type:varchar(30);not null;default:'common'" json:"rarity"`
	Category   string  `gorm:"type:varchar(30);not null;default:'other'" json:"category"`
	GridWidth  int     `gorm:"default:1" json:"grid_width"`
	GridHeight int     `gorm:"default:1" json:"grid_height"`
	EquipSlot  *string `gorm:"type:varchar(30)" json:"equip_slot"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Game               Game                    `gorm:"foreignKey:GameID" json:"game,omitempty"`
	CreatedBy          User                    `gorm:"foreignKey:CreatedByID" json:"created_by,omitempty"`
	Image              *Upload                 `gorm:"foreignKey:ImageID" json:"image,omitempty"`
	Types              []ItemType              `gorm:"foreignKey:ItemID" json:"types,omitempty"`
	Tags               []GameItemTag           `gorm:"many2many:item_tag_assignments;" json:"tags,omitempty"`
	RequiredAttributes []ItemRequiredAttribute `gorm:"foreignKey:ItemID" json:"required_attributes,omitempty"`
	AttributeModifiers []ItemAttributeModifier `gorm:"foreignKey:ItemID" json:"attribute_modifiers,omitempty"`
}
