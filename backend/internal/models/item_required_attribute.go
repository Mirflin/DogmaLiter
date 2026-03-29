package models

type ItemRequiredAttribute struct {
	ID            string `gorm:"type:varchar(36);primaryKey" json:"id"`
	ItemID        string `gorm:"type:varchar(36);not null;index" json:"item_id"`
	AttributeName string `gorm:"type:varchar(50);not null" json:"attribute_name"`
	MinValue      int    `gorm:"not null" json:"min_value"`

	Item Item `gorm:"foreignKey:ItemID" json:"-"`
}