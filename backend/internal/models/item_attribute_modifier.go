package models

type ItemAttributeModifier struct {
	ID            string `gorm:"type:varchar(36);primaryKey" json:"id"`
	ItemID        string `gorm:"type:varchar(36);not null;index" json:"item_id"`
	AttributeName string `gorm:"type:varchar(50);not null" json:"attribute_name"`
	ModifierValue int    `gorm:"not null" json:"modifier_value"`
	IsPercentage  bool   `gorm:"default:false" json:"is_percentage"`

	Item Item `gorm:"foreignKey:ItemID" json:"-"`
}