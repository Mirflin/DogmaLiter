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
			log.Printf("⚠️ FK uploads.user_id: %v (possibly already exists)", err)
		}
	}

	if !db.Migrator().HasConstraint(&models.User{}, "fk_users_avatar") {
		err = db.Exec(`ALTER TABLE users ADD CONSTRAINT fk_users_avatar FOREIGN KEY (avatar_id) REFERENCES uploads(id)`).Error
		if err != nil {
			log.Printf("⚠️ FK users.avatar_id: %v (possibly already exists)", err)
		}
	}

	log.Println("Migration completed")
}
