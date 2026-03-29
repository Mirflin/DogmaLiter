package models

type ItemType struct {
	ID       string `gorm:"type:varchar(36);primaryKey" json:"id"`
	ItemID   string `gorm:"type:varchar(36);not null;index" json:"item_id"`
	TypeName string `gorm:"type:varchar(50);not null" json:"type_name"`

	Item Item `gorm:"foreignKey:ItemID" json:"-"`
}