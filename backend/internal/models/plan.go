package models

import "time"

type Plan struct {
	ID                   string    `gorm:"type:varchar(30);primaryKey" json:"id"`
	Name                 string    `gorm:"type:varchar(100);not null" json:"name"`
	PriceMonthly         float64   `gorm:"type:decimal(10,2);default:0" json:"price_monthly"`
	MaxGamesOwned        int       `gorm:"not null" json:"max_games_owned"`
	MaxPlayersPerGame    int       `gorm:"not null" json:"max_players_per_game"`
	MaxMapsPerGame       int       `gorm:"not null" json:"max_maps_per_game"`
	MaxItemsPerGame      int       `gorm:"not null" json:"max_items_per_game"`
	MaxCharactersPerGame int       `gorm:"not null" json:"max_characters_per_game"`
	MaxUploadSizeMB      int       `gorm:"not null" json:"max_upload_size_mb"`
	StorageLimitMB       int       `gorm:"not null" json:"storage_limit_mb"`
	CreatedAt            time.Time `json:"created_at"`
}
