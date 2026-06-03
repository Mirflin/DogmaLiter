package auth

import (
	"time"

	"gorm.io/gorm"

	"backend/internal/models"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *Repository) AdminStats() (map[string]int64, error) {
	stats := map[string]int64{}
	targets := []struct {
		key   string
		model interface{}
	}{
		{"users", &models.User{}},
		{"games", &models.Game{}},
		{"news", &models.NewsPost{}},
		{"items", &models.Item{}},
		{"characters", &models.Character{}},
	}

	for _, target := range targets {
		var count int64
		if err := r.db.Model(target.model).Count(&count).Error; err != nil {
			return nil, err
		}
		stats[target.key] = count
	}

	return stats, nil
}

func (r *Repository) ListRecentUsers(limit int) ([]models.User, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	var users []models.User
	err := r.db.Preload("Plan").Order("created_at DESC").Limit(limit).Find(&users).Error
	return users, err
}

func (r *Repository) AdminUpdateUser(userID string, updates map[string]interface{}) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
}

func (r *Repository) DeleteUser(userID, actingAdminID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 1. Fully delete games owned by the user (with all their content).
		var ownedGameIDs []string
		if err := tx.Model(&models.Game{}).Where("owner_id = ?", userID).Pluck("id", &ownedGameIDs).Error; err != nil {
			return err
		}
		if len(ownedGameIDs) > 0 {
			var gameCharIDs []string
			if err := tx.Model(&models.Character{}).Where("game_id IN ?", ownedGameIDs).Pluck("id", &gameCharIDs).Error; err != nil {
				return err
			}
			if len(gameCharIDs) > 0 {
				if err := tx.Where("character_id IN ?", gameCharIDs).Delete(&models.CharacterEquipment{}).Error; err != nil {
					return err
				}
				if err := tx.Where("character_id IN ?", gameCharIDs).Delete(&models.CharacterInventory{}).Error; err != nil {
					return err
				}
				if err := tx.Where("character_id IN ?", gameCharIDs).Delete(&models.CharacterCustomAttribute{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("game_id IN ?", ownedGameIDs).Delete(&models.Character{}).Error; err != nil {
				return err
			}

			var gameItemIDs []string
			if err := tx.Model(&models.Item{}).Where("game_id IN ?", ownedGameIDs).Pluck("id", &gameItemIDs).Error; err != nil {
				return err
			}
			if len(gameItemIDs) > 0 {
				if err := tx.Where("item_id IN ?", gameItemIDs).Delete(&models.ItemTagAssignment{}).Error; err != nil {
					return err
				}
				if err := tx.Where("item_id IN ?", gameItemIDs).Delete(&models.ItemRequiredAttribute{}).Error; err != nil {
					return err
				}
				if err := tx.Where("item_id IN ?", gameItemIDs).Delete(&models.ItemAttributeModifier{}).Error; err != nil {
					return err
				}
				if err := tx.Where("item_id IN ?", gameItemIDs).Delete(&models.ItemType{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("game_id IN ?", ownedGameIDs).Delete(&models.Item{}).Error; err != nil {
				return err
			}
			if err := tx.Where("game_id IN ?", ownedGameIDs).Delete(&models.GameItemTag{}).Error; err != nil {
				return err
			}
			if err := tx.Where("game_id IN ?", ownedGameIDs).Delete(&models.GameMap{}).Error; err != nil {
				return err
			}
			if err := tx.Where("game_id IN ?", ownedGameIDs).Delete(&models.ChatMessage{}).Error; err != nil {
				return err
			}
			if err := tx.Where("game_id IN ?", ownedGameIDs).Delete(&models.GameMember{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", ownedGameIDs).Delete(&models.Game{}).Error; err != nil {
				return err
			}
		}

		// 2. Reassign content the user created but that lives in others' games.
		if err := tx.Model(&models.Character{}).Where("created_by_id = ? AND user_id <> ?", userID, userID).
			Update("created_by_id", gorm.Expr("user_id")).Error; err != nil {
			return err
		}
		if err := tx.Exec(`UPDATE items i JOIN games g ON i.game_id = g.id SET i.created_by_id = g.owner_id WHERE i.created_by_id = ?`, userID).Error; err != nil {
			return err
		}

		// 3. Delete characters still owned by the user (in others' games).
		var ownCharIDs []string
		if err := tx.Model(&models.Character{}).Where("user_id = ?", userID).Pluck("id", &ownCharIDs).Error; err != nil {
			return err
		}
		if len(ownCharIDs) > 0 {
			if err := tx.Where("character_id IN ?", ownCharIDs).Delete(&models.CharacterEquipment{}).Error; err != nil {
				return err
			}
			if err := tx.Where("character_id IN ?", ownCharIDs).Delete(&models.CharacterInventory{}).Error; err != nil {
				return err
			}
			if err := tx.Where("character_id IN ?", ownCharIDs).Delete(&models.CharacterCustomAttribute{}).Error; err != nil {
				return err
			}
			if err := tx.Where("user_id = ?", userID).Delete(&models.Character{}).Error; err != nil {
				return err
			}
		}

		// 4. Reassign authored news to the acting admin so it survives.
		if err := tx.Model(&models.NewsPost{}).Where("author_id = ?", userID).Update("author_id", actingAdminID).Error; err != nil {
			return err
		}

		// 5. Memberships, verification tokens, storage usage.
		if err := tx.Where("user_id = ?", userID).Delete(&models.GameMember{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.VerificationToken{}).Error; err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserStorageUsage{}).Error; err != nil {
			return err
		}

		// 6. Detach avatar, reassign uploads to the acting admin (keeps referenced images valid).
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Update("avatar_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Model(&models.Upload{}).Where("user_id = ?", userID).Update("user_id", actingAdminID).Error; err != nil {
			return err
		}

		// 7. The account itself.
		return tx.Delete(&models.User{}, "id = ?", userID).Error
	})
}

func (r *Repository) GetUserByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := r.db.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetUserByID(id string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) VerifyUser(userID string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("is_verified", true).Error
}

func (r *Repository) UpdatePassword(userID string, passwordHash string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("password_hash", passwordHash).Error
}

func (r *Repository) CreateVerificationToken(token *models.VerificationToken) error {
	return r.db.Create(token).Error
}

func (r *Repository) GetVerificationToken(token string, tokenType string) (*models.VerificationToken, error) {
	var vt models.VerificationToken
	err := r.db.Where("token = ? AND type = ? AND expires_at > ?", token, tokenType, time.Now()).First(&vt).Error
	if err != nil {
		return nil, err
	}
	return &vt, nil
}

func (r *Repository) DeleteVerificationToken(id string) error {
	return r.db.Delete(&models.VerificationToken{}, "id = ?", id).Error
}

func (r *Repository) DeleteUserVerificationTokens(userID string, tokenType string) error {
	return r.db.Where("user_id = ? AND type = ?", userID, tokenType).Delete(&models.VerificationToken{}).Error
}

func (r *Repository) UpdateUsername(userID string, username string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("username", username).Error
}

func (r *Repository) UpdateAvatarID(userID string, avatarID *string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("avatar_id", avatarID).Error
}

func (r *Repository) CreateUpload(upload *models.Upload) error {
	return r.db.Create(upload).Error
}

func (r *Repository) GetUploadByID(id string) (*models.Upload, error) {
	var upload models.Upload
	err := r.db.First(&upload, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &upload, nil
}

func (r *Repository) DeleteUpload(id string) error {
	return r.db.Delete(&models.Upload{}, "id = ?", id).Error
}

func (r *Repository) GetStorageUsage(userID string) (*models.UserStorageUsage, error) {
	var usage models.UserStorageUsage
	err := r.db.First(&usage, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &usage, nil
}

func (r *Repository) CreateStorageUsage(userID string) error {
	return r.db.Create(&models.UserStorageUsage{UserID: userID}).Error
}

func (r *Repository) AddStorageUsage(userID string, bytes int64) error {
	return r.db.Model(&models.UserStorageUsage{}).Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"used_bytes":  gorm.Expr("used_bytes + ?", bytes),
			"files_count": gorm.Expr("files_count + 1"),
		}).Error
}

func (r *Repository) SubtractStorageUsage(userID string, bytes int64) error {
	return r.db.Model(&models.UserStorageUsage{}).Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"used_bytes":  gorm.Expr("GREATEST(used_bytes - ?, 0)", bytes),
			"files_count": gorm.Expr("GREATEST(files_count - 1, 0)"),
		}).Error
}

func (r *Repository) GetUserWithPlan(userID string) (*models.User, error) {
	var user models.User
	err := r.db.Preload("Plan").First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *Repository) GetAllPlans() ([]models.Plan, error) {
	var plans []models.Plan
	err := r.db.Order("price_monthly ASC").Find(&plans).Error
	if err != nil {
		return nil, err
	}
	return plans, nil
}
