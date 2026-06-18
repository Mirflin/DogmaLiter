package database

import (
	"log"

	"gorm.io/gorm"

	"backend/internal/models"
)

func AutoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(
		&models.Plan{},
		&models.User{},
		&models.Upload{},
		&models.UserStorageUsage{},
		&models.VerificationToken{},
		&models.NewsPost{},
		&models.Game{},
		&models.GameMember{},
		&models.Character{},
		&models.CharacterCustomAttribute{},
		&models.Item{},
		&models.GameItemTag{},
		&models.ItemTagAssignment{},
		&models.ItemType{},
		&models.ItemRequiredAttribute{},
		&models.ItemAttributeModifier{},
		&models.CharacterInventory{},
		&models.CharacterEquipment{},
		&models.GameMap{},
		&models.ChatMessage{},
		&models.ActivityLog{},
		&models.TradeOffer{},
		&models.TradeOfferItem{},
	)
	if err != nil {
		log.Fatalf("Error with migration (stage 1): %v", err)
	}

	migrateItemSchema(db)
	migrateGameStandardAttrs(db)

	if !db.Migrator().HasConstraint(&models.Upload{}, "fk_uploads_user") {
		err = db.Exec(`ALTER TABLE uploads ADD CONSTRAINT fk_uploads_user FOREIGN KEY (user_id) REFERENCES users(id)`).Error
		if err != nil {
			log.Printf("FK uploads.user_id: %v (possibly already exists)", err)
		}
	}

	if !db.Migrator().HasConstraint(&models.User{}, "fk_users_avatar") {
		err = db.Exec(`ALTER TABLE users ADD CONSTRAINT fk_users_avatar FOREIGN KEY (avatar_id) REFERENCES uploads(id)`).Error
		if err != nil {
			log.Printf("FK users.avatar_id: %v (possibly already exists)", err)
		}
	}

	seedPlans(db)

	log.Println("Migration completed")
}

func seedPlans(db *gorm.DB) {
	plans := []models.Plan{
		{
			ID:                   "free",
			Name:                 "Free",
			PriceMonthly:         0,
			MaxGamesOwned:        2,
			MaxPlayersPerGame:    5,
			MaxMapsPerGame:       3,
			MaxItemsPerGame:      20,
			MaxCharactersPerGame: 5,
			MaxUploadSizeMB:      5,
			StorageLimitMB:       100,
		},
		{
			ID:                   "plus",
			Name:                 "Plus",
			PriceMonthly:         4.99,
			MaxGamesOwned:        10,
			MaxPlayersPerGame:    15,
			MaxMapsPerGame:       15,
			MaxItemsPerGame:      100,
			MaxCharactersPerGame: 20,
			MaxUploadSizeMB:      25,
			StorageLimitMB:       1024,
		},
		{
			ID:                   "pro",
			Name:                 "Pro",
			PriceMonthly:         9.99,
			MaxGamesOwned:        -1,
			MaxPlayersPerGame:    -1,
			MaxMapsPerGame:       -1,
			MaxItemsPerGame:      -1,
			MaxCharactersPerGame: -1,
			MaxUploadSizeMB:      50,
			StorageLimitMB:       5120,
		},
	}

	for _, plan := range plans {
		var existing models.Plan
		if err := db.First(&existing, "id = ?", plan.ID).Error; err != nil {
			db.Create(&plan)
			log.Printf("Seeded plan: %s", plan.Name)
		}
	}
}

// migrateGameStandardAttrs preserves prior behaviour after the per-attribute
// toggle column was introduced: games that had standard attributes turned off
// (the old single boolean) start with an empty enabled list. Newly added column
// defaults to all six, so this only adjusts the disabled ones. Idempotent.
func migrateGameStandardAttrs(db *gorm.DB) {
	if !db.Migrator().HasColumn(&models.Game{}, "enabled_standard_attrs") {
		return
	}
	if err := db.Exec(`UPDATE games SET enabled_standard_attrs = '' WHERE show_standard_attrs = 0`).Error; err != nil {
		log.Printf("games.enabled_standard_attrs backfill: %v", err)
	}
}

func migrateItemSchema(db *gorm.DB) {
	if !db.Migrator().HasTable(&models.Item{}) {
		return
	}

	if err := db.Exec(`ALTER TABLE items MODIFY COLUMN rarity VARCHAR(30) NOT NULL DEFAULT 'common'`).Error; err != nil {
		log.Printf("items.rarity migration: %v", err)
	}

	if !db.Migrator().HasColumn(&models.Item{}, "category") {
		if err := db.Exec(`ALTER TABLE items ADD COLUMN category VARCHAR(30) NOT NULL DEFAULT 'other' AFTER rarity`).Error; err != nil {
			log.Printf("items.category migration: %v", err)
		}
	}

	if err := db.Exec(`UPDATE items SET rarity = 'unique' WHERE rarity = 'artifact'`).Error; err != nil {
		log.Printf("items.rarity backfill: %v", err)
	}

	if err := db.Exec(`UPDATE items SET equip_slot = 'ring' WHERE equip_slot IN ('ring_1', 'ring_2')`).Error; err != nil {
		log.Printf("items.ring slot backfill: %v", err)
	}

	if err := db.Exec(`
		UPDATE items
		SET category = 'consumable'
		WHERE category = 'other'
			AND EXISTS (
				SELECT 1
				FROM item_types
				WHERE item_types.item_id = items.id
					AND LOWER(item_types.type_name) = 'consumable'
			)
	`).Error; err != nil {
		log.Printf("items.category consumable backfill: %v", err)
	}

	if err := db.Exec(`UPDATE items SET category = 'equipment' WHERE category IN ('other', 'loot') AND COALESCE(equip_slot, '') <> ''`).Error; err != nil {
		log.Printf("items.category equipment backfill: %v", err)
	}
}
