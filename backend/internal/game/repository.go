package game

import (
	"strings"
	"time"

	"backend/internal/models"

	"github.com/google/uuid"
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

func (r *Repository) UpdateCharacter(character *models.Character, replaceCustomAttributes bool) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Character{}).
			Where("game_id = ? AND id = ?", character.GameID, character.ID).
			Updates(map[string]interface{}{
				"user_id":           character.UserID,
				"name":              character.Name,
				"backstory":         character.Backstory,
				"base_strength":     character.BaseStrength,
				"base_dexterity":    character.BaseDexterity,
				"base_constitution": character.BaseConstitution,
				"base_intelligence": character.BaseIntelligence,
				"base_wisdom":       character.BaseWisdom,
				"base_charisma":     character.BaseCharisma,
				"inventory_width":   character.InventoryWidth,
				"inventory_height":  character.InventoryHeight,
				"currency_gold":     character.CurrencyGold,
				"currency_silver":   character.CurrencySilver,
				"currency_copper":   character.CurrencyCopper,
				"updated_at":        time.Now(),
			}).Error; err != nil {
			return err
		}

		if !replaceCustomAttributes {
			return nil
		}

		if err := tx.Where("character_id = ?", character.ID).Delete(&models.CharacterCustomAttribute{}).Error; err != nil {
			return err
		}

		if len(character.CustomAttributes) == 0 {
			return nil
		}

		return tx.Create(&character.CustomAttributes).Error
	})
}

func (r *Repository) AddCharacterInventoryItems(entries []models.CharacterInventory) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.Create(&entries).Error
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
		Preload("Tags", func(db *gorm.DB) *gorm.DB {
			return db.Order("name ASC")
		}).
		Preload("RequiredAttributes").
		Preload("AttributeModifiers").
		Order("updated_at DESC, created_at DESC").
		Find(&items).Error

	return items, err
}

func (r *Repository) ListGameItemsPage(gameID string, params listGameItemsParams) ([]models.Item, int64, error) {
	baseQuery := r.db.Model(&models.Item{}).
		Where("items.game_id = ?", gameID)

	if params.Tag != "" || params.Search != "" {
		baseQuery = baseQuery.
			Joins("LEFT JOIN item_tag_assignments ON item_tag_assignments.item_id = items.id").
			Joins("LEFT JOIN game_item_tags ON game_item_tags.id = item_tag_assignments.game_item_tag_id")
	}

	if params.Search != "" {
		searchTerm := "%" + strings.ToLower(params.Search) + "%"
		baseQuery = baseQuery.Where(`
			LOWER(items.name) LIKE ?
			OR LOWER(COALESCE(items.description, '')) LIKE ?
			OR LOWER(items.category) LIKE ?
			OR LOWER(COALESCE(items.equip_slot, '')) LIKE ?
			OR LOWER(COALESCE(game_item_tags.name, '')) LIKE ?
		`, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	}

	if params.Rarity != "" {
		baseQuery = baseQuery.Where("items.rarity = ?", params.Rarity)
	}
	if params.Category != "" {
		baseQuery = baseQuery.Where("items.category = ?", params.Category)
	}
	if params.Slot != "" {
		baseQuery = baseQuery.Where("items.equip_slot = ?", params.Slot)
	}
	if params.Tag != "" {
		baseQuery = baseQuery.Where("LOWER(game_item_tags.name) = LOWER(?)", params.Tag)
	}

	var totalItems int64
	if err := baseQuery.Distinct("items.id").Count(&totalItems).Error; err != nil {
		return nil, 0, err
	}
	if totalItems == 0 {
		return []models.Item{}, 0, nil
	}

	// Group by the item primary key so joined tag rows collapse without forcing
	// a DISTINCT select that MySQL refuses to sort by non-selected item columns.
	itemIDsQuery := applyItemListSort(baseQuery.Select("items.id").Group("items.id"), params.Sort).
		Limit(params.PerPage).
		Offset((params.Page - 1) * params.PerPage)

	var itemIDs []string
	if err := itemIDsQuery.Pluck("items.id", &itemIDs).Error; err != nil {
		return nil, 0, err
	}
	if len(itemIDs) == 0 {
		return []models.Item{}, totalItems, nil
	}

	var items []models.Item
	err := r.db.
		Where("game_id = ? AND id IN ?", gameID, itemIDs).
		Preload("Image").
		Preload("Types").
		Preload("Tags", func(db *gorm.DB) *gorm.DB {
			return db.Order("name ASC")
		}).
		Preload("RequiredAttributes").
		Preload("AttributeModifiers").
		Find(&items).Error
	if err != nil {
		return nil, 0, err
	}

	itemsByID := make(map[string]models.Item, len(items))
	for _, item := range items {
		itemsByID[item.ID] = item
	}

	orderedItems := make([]models.Item, 0, len(itemIDs))
	for _, itemID := range itemIDs {
		item, ok := itemsByID[itemID]
		if !ok {
			continue
		}
		orderedItems = append(orderedItems, item)
	}

	return orderedItems, totalItems, nil
}

func (r *Repository) CountGameItems(gameID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Item{}).
		Where("game_id = ?", gameID).
		Count(&count).Error
	return count, err
}

func (r *Repository) GetItemByID(gameID, itemID string) (*models.Item, error) {
	var item models.Item
	err := r.db.
		Where("game_id = ? AND id = ?", gameID, itemID).
		Preload("Image").
		Preload("Types").
		Preload("Tags", func(db *gorm.DB) *gorm.DB {
			return db.Order("name ASC")
		}).
		Preload("RequiredAttributes").
		Preload("AttributeModifiers").
		First(&item).Error
	if err != nil {
		return nil, err
	}

	return &item, nil
}

func (r *Repository) CreateItem(item *models.Item, tagNames []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("Game", "CreatedBy", "Image", "Types", "Tags", "RequiredAttributes", "AttributeModifiers").Create(item).Error; err != nil {
			return err
		}

		if len(item.RequiredAttributes) > 0 {
			if err := tx.Create(&item.RequiredAttributes).Error; err != nil {
				return err
			}
		}

		if len(item.AttributeModifiers) > 0 {
			if err := tx.Create(&item.AttributeModifiers).Error; err != nil {
				return err
			}
		}

		if len(tagNames) > 0 {
			tags, err := r.ensureGameItemTagsTx(tx, item.GameID, item.CreatedByID, tagNames)
			if err != nil {
				return err
			}

			if err := tx.Model(item).Association("Tags").Replace(tags); err != nil {
				return err
			}
		}

		return nil
	})
}

func (r *Repository) UpdateItem(item *models.Item, tagNames []string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Item{}).
			Where("game_id = ? AND id = ?", item.GameID, item.ID).
			Updates(map[string]interface{}{
				"name":        item.Name,
				"description": item.Description,
				"rarity":      item.Rarity,
				"category":    item.Category,
				"grid_width":  item.GridWidth,
				"grid_height": item.GridHeight,
				"equip_slot":  item.EquipSlot,
				"updated_at":  time.Now(),
			}).Error; err != nil {
			return err
		}

		if err := tx.Where("item_id = ?", item.ID).Delete(&models.ItemRequiredAttribute{}).Error; err != nil {
			return err
		}
		if len(item.RequiredAttributes) > 0 {
			if err := tx.Create(&item.RequiredAttributes).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("item_id = ?", item.ID).Delete(&models.ItemAttributeModifier{}).Error; err != nil {
			return err
		}
		if len(item.AttributeModifiers) > 0 {
			if err := tx.Create(&item.AttributeModifiers).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("item_id = ?", item.ID).Delete(&models.ItemType{}).Error; err != nil {
			return err
		}

		tags := make([]models.GameItemTag, 0, len(tagNames))
		if len(tagNames) > 0 {
			resolvedTags, err := r.ensureGameItemTagsTx(tx, item.GameID, item.CreatedByID, tagNames)
			if err != nil {
				return err
			}
			tags = resolvedTags
		}

		if err := tx.Model(item).Association("Tags").Replace(tags); err != nil {
			return err
		}

		return nil
	})
}

func (r *Repository) ListGameItemTags(gameID string) ([]models.GameItemTag, error) {
	var tags []models.GameItemTag
	err := r.db.
		Where("game_id = ?", gameID).
		Order("name ASC").
		Find(&tags).Error

	return tags, err
}

func (r *Repository) ensureGameItemTagsTx(tx *gorm.DB, gameID, userID string, tagNames []string) ([]models.GameItemTag, error) {
	tags := make([]models.GameItemTag, 0, len(tagNames))
	seen := make(map[string]struct{}, len(tagNames))

	for _, rawName := range tagNames {
		name := strings.TrimSpace(rawName)
		if name == "" {
			continue
		}

		lookupKey := strings.ToLower(name)
		if _, exists := seen[lookupKey]; exists {
			continue
		}
		seen[lookupKey] = struct{}{}

		var tag models.GameItemTag
		err := tx.Where("game_id = ? AND LOWER(name) = LOWER(?)", gameID, name).First(&tag).Error
		if err != nil {
			if err != gorm.ErrRecordNotFound {
				return nil, err
			}

			tag = models.GameItemTag{
				ID:          uuid.New().String(),
				GameID:      gameID,
				CreatedByID: userID,
				Name:        name,
			}

			if err := tx.Create(&tag).Error; err != nil {
				return nil, err
			}
		}

		tags = append(tags, tag)
	}

	return tags, nil
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

func (r *Repository) DeleteItem(gameID, itemID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var inventoryItemIDs []string
		if err := tx.Model(&models.CharacterInventory{}).
			Where("item_id = ?", itemID).
			Pluck("id", &inventoryItemIDs).Error; err != nil {
			return err
		}

		if len(inventoryItemIDs) > 0 {
			if err := tx.Where("inventory_item_id IN ?", inventoryItemIDs).Delete(&models.CharacterEquipment{}).Error; err != nil {
				return err
			}
		}

		if err := tx.Where("item_id = ?", itemID).Delete(&models.CharacterInventory{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id = ?", itemID).Delete(&models.ItemTagAssignment{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id = ?", itemID).Delete(&models.ItemRequiredAttribute{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id = ?", itemID).Delete(&models.ItemAttributeModifier{}).Error; err != nil {
			return err
		}
		if err := tx.Where("item_id = ?", itemID).Delete(&models.ItemType{}).Error; err != nil {
			return err
		}

		return tx.Where("game_id = ? AND id = ?", gameID, itemID).Delete(&models.Item{}).Error
	})
}

func (r *Repository) UpdateCoverImage(gameID string, coverImageID *string) error {
	return r.db.Model(&models.Game{}).Where("id = ?", gameID).Update("cover_image_id", coverImageID).Error
}

func (r *Repository) UpdateItemImage(gameID, itemID string, imageID *string) error {
	return r.db.Model(&models.Item{}).
		Where("game_id = ? AND id = ?", gameID, itemID).
		Updates(map[string]interface{}{
			"image_id":   imageID,
			"updated_at": time.Now(),
		}).Error
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

func applyItemListSort(query *gorm.DB, sort string) *gorm.DB {
	switch sort {
	case "name-asc":
		return query.Order("items.name ASC").Order("items.updated_at DESC").Order("items.id ASC")
	case "name-desc":
		return query.Order("items.name DESC").Order("items.updated_at DESC").Order("items.id DESC")
	case "rarity":
		return query.Order(`CASE items.rarity
			WHEN 'common' THEN 1
			WHEN 'uncommon' THEN 2
			WHEN 'rare' THEN 3
			WHEN 'epic' THEN 4
			WHEN 'masterwork' THEN 5
			WHEN 'legendary' THEN 6
			WHEN 'unique' THEN 7
			ELSE 0 END DESC`).
			Order("items.name ASC").
			Order("items.id ASC")
	case "size":
		return query.Order("(items.grid_width * items.grid_height) DESC").Order("items.name ASC").Order("items.id ASC")
	default:
		return query.Order("items.updated_at DESC").Order("items.created_at DESC").Order("items.id DESC")
	}
}
