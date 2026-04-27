package game

import (
	"time"

	"backend/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) GetUserGames(userID string) ([]models.Game, error) {
	var games []models.Game

	err := r.db.
		Distinct("games.*").
		Joins("LEFT JOIN game_members ON game_members.game_id = games.id").
		Where("games.owner_id = ? OR game_members.user_id = ?", userID, userID).
		Preload("CoverImage").
		Preload("Members").
		Preload("Members.User").
		Order("games.updated_at DESC").
		Find(&games).Error

	return games, err
}

func (r *Repository) CountUserGames(userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Game{}).
		Distinct("games.id").
		Joins("LEFT JOIN game_members ON game_members.game_id = games.id").
		Where("games.owner_id = ? OR game_members.user_id = ?", userID, userID).
		Count(&count).Error
	return count, err
}

func (r *Repository) CreateGame(game *models.Game) error {
	return r.db.Create(game).Error
}

func (r *Repository) AddMember(member *models.GameMember) error {
	return r.db.Create(member).Error
}

func (r *Repository) GetGameByInviteCode(code string) (*models.Game, error) {
	var game models.Game
	err := r.db.Where("invite_code = ?", code).First(&game).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func (r *Repository) GetGameByID(id string) (*models.Game, error) {
	var game models.Game
	err := r.db.Preload("Members").Preload("Members.User").Preload("Owner").Preload("Owner.Plan").Preload("CoverImage").First(&game, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &game, nil
}

func (r *Repository) IsMember(gameID, userID string) (bool, error) {
	var count int64
	err := r.db.Model(&models.GameMember{}).
		Where("game_id = ? AND user_id = ?", gameID, userID).
		Count(&count).Error
	return count > 0, err
}

func (r *Repository) ListGameCharacters(gameID string, userID *string) ([]models.Character, error) {
	var characters []models.Character

	query := r.db.Where("game_id = ?", gameID)
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	err := query.
		Preload("User").
		Preload("Portrait").
		Preload("CustomAttributes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, created_at ASC")
		}).
		Order("updated_at DESC, created_at DESC").
		Find(&characters).Error

	return characters, err
}

func (r *Repository) GetCharacterByID(gameID, characterID string) (*models.Character, error) {
	var character models.Character

	err := r.db.
		Where("game_id = ? AND id = ?", gameID, characterID).
		Preload("User").
		Preload("Portrait").
		Preload("CustomAttributes", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort_order ASC, created_at ASC")
		}).
		Preload("Inventory", func(db *gorm.DB) *gorm.DB {
			return db.Order("grid_y ASC, grid_x ASC, created_at ASC")
		}).
		Preload("Inventory.Item").
		Preload("Inventory.Item.Image").
		Preload("Inventory.Item.Types").
		Preload("Inventory.Item.RequiredAttributes").
		Preload("Inventory.Item.AttributeModifiers").
		Preload("Equipment", func(db *gorm.DB) *gorm.DB {
			return db.Order("slot ASC")
		}).
		Preload("Equipment.InventoryItem").
		Preload("Equipment.InventoryItem.Item").
		Preload("Equipment.InventoryItem.Item.Image").
		Preload("Equipment.InventoryItem.Item.Types").
		Preload("Equipment.InventoryItem.Item.RequiredAttributes").
		Preload("Equipment.InventoryItem.Item.AttributeModifiers").
		First(&character).Error
	if err != nil {
		return nil, err
	}

	return &character, nil
}

func (r *Repository) CountGameCharactersForUser(gameID, userID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Character{}).
		Where("game_id = ? AND user_id = ?", gameID, userID).
		Count(&count).Error
	return count, err
}

func (r *Repository) CreateCharacter(character *models.Character) error {
	return r.db.Create(character).Error
}

func (r *Repository) UpdateCharacter(character *models.Character) error {
	return r.db.Model(&models.Character{}).
		Where("game_id = ? AND id = ?", character.GameID, character.ID).
		Updates(map[string]interface{}{
			"name":            character.Name,
			"backstory":       character.Backstory,
			"currency_gold":   character.CurrencyGold,
			"currency_silver": character.CurrencySilver,
			"currency_copper": character.CurrencyCopper,
			"updated_at":      time.Now(),
		}).Error
}

func (r *Repository) UpdateCharacterPortrait(gameID, characterID string, portraitID *string) error {
	return r.db.Model(&models.Character{}).
		Where("game_id = ? AND id = ?", gameID, characterID).
		Updates(map[string]interface{}{
			"portrait_id": portraitID,
			"updated_at":  time.Now(),
		}).Error
}

func (r *Repository) ListGameItems(gameID string) ([]models.Item, error) {
	var items []models.Item

	err := r.db.
		Where("game_id = ?", gameID).
		Preload("Image").
		Preload("Types").
		Preload("RequiredAttributes").
		Preload("AttributeModifiers").
		Order("updated_at DESC, created_at DESC").
		Find(&items).Error

	return items, err
}

func (r *Repository) ListGameChatMessages(gameID string, limit int) ([]models.ChatMessage, error) {
	if limit <= 0 || limit > 40 {
		limit = 40
	}

	var messages []models.ChatMessage
	err := r.db.
		Where("game_id = ?", gameID).
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error
	if err != nil {
		return nil, err
	}

	for left, right := 0, len(messages)-1; left < right; left, right = left+1, right-1 {
		messages[left], messages[right] = messages[right], messages[left]
	}

	return messages, nil
}

func (r *Repository) CreateChatMessage(message *models.ChatMessage) error {
	return r.db.Create(message).Error
}

func (r *Repository) TrimGameChatMessages(gameID string, keep int) error {
	if keep <= 0 {
		keep = 40
	}

	var staleIDs []string
	if err := r.db.Model(&models.ChatMessage{}).
		Where("game_id = ?", gameID).
		Order("created_at DESC, id DESC").
		Offset(keep).
		Pluck("id", &staleIDs).Error; err != nil {
		return err
	}

	if len(staleIDs) == 0 {
		return nil
	}

	return r.db.Where("game_id = ? AND id IN ?", gameID, staleIDs).Delete(&models.ChatMessage{}).Error
}

func (r *Repository) GetUserPlan(userID string) (*models.Plan, error) {
	var user models.User
	err := r.db.Preload("Plan").First(&user, "id = ?", userID).Error
	if err != nil {
		return nil, err
	}
	return &user.Plan, nil
}

func (r *Repository) UpdateInviteCode(gameID, code string, expiresAt time.Time) error {
	return r.db.Model(&models.Game{}).Where("id = ?", gameID).Updates(map[string]interface{}{
		"invite_code":            code,
		"invite_code_expires_at": expiresAt,
	}).Error
}

func (r *Repository) MemberCount(gameID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.GameMember{}).Where("game_id = ?", gameID).Count(&count).Error
	return count, err
}

func (r *Repository) RemoveMember(gameID, userID string) error {
	return r.db.Where("game_id = ? AND user_id = ?", gameID, userID).Delete(&models.GameMember{}).Error
}

func (r *Repository) DeleteGame(gameID string) error {
	if err := r.db.Where("game_id = ?", gameID).Delete(&models.GameMember{}).Error; err != nil {
		return err
	}
	return r.db.Delete(&models.Game{}, "id = ?", gameID).Error
}

func (r *Repository) UpdateCoverImage(gameID string, coverImageID *string) error {
	return r.db.Model(&models.Game{}).Where("id = ?", gameID).Update("cover_image_id", coverImageID).Error
}

func (r *Repository) UpdateGame(game *models.Game) error {
	return r.db.Save(game).Error
}

func (r *Repository) CreateUpload(upload *models.Upload) error {
	return r.db.Create(upload).Error
}

func (r *Repository) GetUploadByID(id string) (*models.Upload, error) {
	var upload models.Upload
	err := r.db.First(&upload, "id = ?", id).Error
	return &upload, err
}

func (r *Repository) DeleteUpload(id string) error {
	return r.db.Delete(&models.Upload{}, "id = ?", id).Error
}

func (r *Repository) GetStorageUsage(userID string) (*models.UserStorageUsage, error) {
	var usage models.UserStorageUsage
	err := r.db.First(&usage, "user_id = ?", userID).Error
	return &usage, err
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
