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
		&models.ItemType{},
		&models.ItemRequiredAttribute{},
		&models.ItemAttributeModifier{},
		&models.CharacterInventory{},
		&models.CharacterEquipment{},
		&models.GameMap{},
		&models.ChatMessage{},
	)
	if err != nil {
		log.Fatalf("Error with migration (stage 1): %v", err)
	}

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
