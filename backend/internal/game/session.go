package game

import (
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"backend/internal/auth"
	"backend/internal/models"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var errGameAccessDenied = errors.New("game access denied")

const nonGMCharacterLimit = 5
const gameChatMessageLimit = 40

func (h *Handler) GetSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	var characterOwnerID *string
	if !isGM {
		characterOwnerID = &userID
	}

	characters, err := h.service.repo.ListGameCharacters(gameID, characterOwnerID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load characters"})
		return
	}

	ownedCharacterCount := len(characters)
	if isGM {
		count, err := h.service.repo.CountGameCharactersForUser(gameID, userID)
		if err != nil {
			respondJSON(w, 500, map[string]string{"error": "Failed to load character limit"})
			return
		}
		ownedCharacterCount = int(count)
	}

	items, err := h.service.repo.ListGameItems(gameID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load items"})
		return
	}

	messages := make([]models.ChatMessage, 0)
	if game.EnableChat {
		messages, err = h.service.repo.ListGameChatMessages(gameID, gameChatMessageLimit)
		if err != nil {
			respondJSON(w, 500, map[string]string{"error": "Failed to load chat messages"})
			return
		}
	}

	respondJSON(w, 200, map[string]interface{}{
		"viewer": map[string]interface{}{
			"user_id":               userID,
			"is_gm":                 isGM,
			"character_limit":       viewerCharacterLimit(isGM),
			"owned_character_count": ownedCharacterCount,
			"can_create_character":  isGM || ownedCharacterCount < nonGMCharacterLimit,
		},
		"game":       serializeGame(game, userID, isGM),
		"characters": serializeCharacterSummaries(characters),
		"items":      serializeItems(items),
		"messages":   serializeChatMessages(messages),
	})
}

func (h *Handler) CreateCharacter(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	if !isGM {
		count, err := h.service.repo.CountGameCharactersForUser(gameID, userID)
		if err != nil {
			respondJSON(w, 500, map[string]string{"error": "Failed to check character limit"})
			return
		}
		if count >= nonGMCharacterLimit {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("Character limit reached (%d)", nonGMCharacterLimit)})
			return
		}
	}

	character := &models.Character{
		ID:               uuid.New().String(),
		GameID:           gameID,
		UserID:           userID,
		CreatedByID:      userID,
		Name:             generateRandomCharacterName(),
		Backstory:        "",
		PortraitID:       nil,
		BaseStrength:     10,
		BaseDexterity:    10,
		BaseConstitution: 10,
		BaseIntelligence: 10,
		BaseWisdom:       10,
		BaseCharisma:     10,
		InventoryWidth:   10,
		InventoryHeight:  6,
		CurrencyGold:     0,
		CurrencySilver:   0,
		CurrencyCopper:   0,
	}

	if err := h.service.repo.CreateCharacter(character); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to create character"})
		return
	}

	createdCharacter, err := h.service.repo.GetCharacterByID(gameID, character.ID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Character was created but could not be loaded"})
		return
	}

	respondJSON(w, 201, map[string]interface{}{
		"character": serializeCharacterDetail(createdCharacter),
	})
}

func (h *Handler) GetCharacter(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	characterID := chi.URLParam(r, "characterID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	character, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": fmt.Sprintf("Character %s not found", characterID)})
		return
	}

	if !isGM && character.UserID != userID {
		respondJSON(w, 403, map[string]string{"error": "You do not have access to this character"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"character": serializeCharacterDetail(character),
	})
}

func (h *Handler) UpdateCharacter(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	characterID := chi.URLParam(r, "characterID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	character, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": fmt.Sprintf("Character %s not found", characterID)})
		return
	}

	if !isGM && character.UserID != userID {
		respondJSON(w, 403, map[string]string{"error": "You do not have access to update this character"})
		return
	}

	var req struct {
		Name           *string `json:"name"`
		Backstory      *string `json:"backstory"`
		CurrencyGold   *int    `json:"currency_gold"`
		CurrencySilver *int    `json:"currency_silver"`
		CurrencyCopper *int    `json:"currency_copper"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	hasChanges := false

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			respondJSON(w, 400, map[string]string{"error": "Character name cannot be empty"})
			return
		}
		if len(name) > 100 {
			respondJSON(w, 400, map[string]string{"error": "Character name must be 100 characters or less"})
			return
		}
		character.Name = name
		hasChanges = true
	}

	if req.Backstory != nil {
		backstory := strings.TrimSpace(*req.Backstory)
		if len(backstory) > 5000 {
			respondJSON(w, 400, map[string]string{"error": "Backstory must be 5000 characters or less"})
			return
		}
		character.Backstory = backstory
		hasChanges = true
	}

	if req.CurrencyGold != nil {
		value, err := validateCurrencyAmount(*req.CurrencyGold, "Gold")
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		character.CurrencyGold = value
		hasChanges = true
	}
	if req.CurrencySilver != nil {
		value, err := validateCurrencyAmount(*req.CurrencySilver, "Silver")
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		character.CurrencySilver = value
		hasChanges = true
	}
	if req.CurrencyCopper != nil {
		value, err := validateCurrencyAmount(*req.CurrencyCopper, "Copper")
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		character.CurrencyCopper = value
		hasChanges = true
	}

	if hasChanges {
		if err := h.service.repo.UpdateCharacter(character); err != nil {
			respondJSON(w, 500, map[string]string{"error": "Failed to update character"})
			return
		}

		character, err = h.service.repo.GetCharacterByID(gameID, characterID)
		if err != nil {
			respondJSON(w, 500, map[string]string{"error": "Character was updated but could not be reloaded"})
			return
		}
	}

	respondJSON(w, 200, map[string]interface{}{
		"character": serializeCharacterDetail(character),
	})
}

func (h *Handler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, _, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	if !game.EnableChat {
		respondJSON(w, 200, map[string]interface{}{
			"enabled":  false,
			"messages": []interface{}{},
		})
		return
	}

	messages, err := h.service.repo.ListGameChatMessages(gameID, gameChatMessageLimit)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load chat messages"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"enabled":  true,
		"messages": serializeChatMessages(messages),
	})
}

func (h *Handler) CreateChatMessage(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, _, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	if !game.EnableChat {
		respondJSON(w, 403, map[string]string{"error": "Chat is disabled for this game"})
		return
	}

	var req struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		respondJSON(w, 400, map[string]string{"error": "Message cannot be empty"})
		return
	}
	if len(content) > 2000 {
		respondJSON(w, 400, map[string]string{"error": "Message must be 2000 characters or less"})
		return
	}

	message := &models.ChatMessage{
		ID:          uuid.New().String(),
		GameID:      gameID,
		UserID:      userID,
		MessageType: "text",
		Content:     content,
		CreatedAt:   time.Now(),
	}

	if err := h.service.repo.CreateChatMessage(message); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to send message"})
		return
	}

	_ = h.service.repo.TrimGameChatMessages(gameID, gameChatMessageLimit)

	respondJSON(w, 201, map[string]interface{}{
		"message": serializeChatMessage(message, findGameMember(game, userID)),
	})
}

func (h *Handler) authorizeGameAccess(userID, gameID string) (*models.Game, bool, error) {
	game, err := h.service.repo.GetGameByID(gameID)
	if err != nil {
		return nil, false, err
	}

	isMember, err := h.service.repo.IsMember(gameID, userID)
	if err != nil {
		return nil, false, err
	}
	if game.OwnerID != userID && !isMember {
		return nil, false, errGameAccessDenied
	}

	isGM := game.OwnerID == userID
	if !isGM {
		for _, member := range game.Members {
			if member.UserID == userID && (member.Role == "gm" || member.Role == "assistant_gm") {
				isGM = true
				break
			}
		}
	}

	return game, isGM, nil
}

func (h *Handler) respondGameAccessError(w http.ResponseWriter, gameID string, err error) {
	if errors.Is(err, errGameAccessDenied) {
		respondJSON(w, 403, map[string]string{"error": fmt.Sprintf("User does not have access to game %s", gameID)})
		return
	}

	respondJSON(w, 404, map[string]string{"error": fmt.Sprintf("Game %s not found", gameID)})
}

func serializeGame(game *models.Game, userID string, isGM bool) map[string]interface{} {
	members := make([]map[string]interface{}, 0, len(game.Members))
	for _, member := range game.Members {
		members = append(members, map[string]interface{}{
			"user_id":   member.UserID,
			"role":      member.Role,
			"username":  member.User.Username,
			"avatar_id": member.User.AvatarID,
			"joined_at": member.JoinedAt,
		})
	}

	payload := map[string]interface{}{
		"id":                  game.ID,
		"title":               game.Title,
		"description":         game.Description,
		"system":              game.System,
		"max_players":         game.MaxPlayers,
		"owner_id":            game.OwnerID,
		"cover_image_id":      game.CoverImageID,
		"show_standard_attrs": game.ShowStandardAttrs,
		"enable_chat":         game.EnableChat,
		"enable_item_trading": game.EnableItemTrading,
		"created_at":          game.CreatedAt,
		"updated_at":          game.UpdatedAt,
		"members":             members,
		"owner": map[string]interface{}{
			"id":         game.Owner.ID,
			"username":   game.Owner.Username,
			"avatar_id":  game.Owner.AvatarID,
			"plan_name":  game.Owner.Plan.Name,
			"created_at": game.Owner.CreatedAt,
		},
		"viewer_user_id": userID,
		"viewer_is_gm":   isGM,
	}

	if isGM {
		payload["invite_code"] = game.InviteCode
		payload["invite_code_expires_at"] = game.InviteCodeExpiresAt
	}

	return payload
}

func serializeCharacterSummaries(characters []models.Character) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(characters))
	for _, character := range characters {
		result = append(result, map[string]interface{}{
			"id":               character.ID,
			"game_id":          character.GameID,
			"user_id":          character.UserID,
			"name":             character.Name,
			"backstory":        character.Backstory,
			"portrait_id":      character.PortraitID,
			"inventory_width":  character.InventoryWidth,
			"inventory_height": character.InventoryHeight,
			"currency_gold":    character.CurrencyGold,
			"currency_silver":  character.CurrencySilver,
			"currency_copper":  character.CurrencyCopper,
			"owner": map[string]interface{}{
				"id":         character.User.ID,
				"username":   character.User.Username,
				"avatar_id":  character.User.AvatarID,
				"created_at": character.User.CreatedAt,
			},
			"base_attributes": map[string]int{
				"strength":     character.BaseStrength,
				"dexterity":    character.BaseDexterity,
				"constitution": character.BaseConstitution,
				"intelligence": character.BaseIntelligence,
				"wisdom":       character.BaseWisdom,
				"charisma":     character.BaseCharisma,
			},
			"custom_attributes": serializeCharacterCustomAttributes(character.CustomAttributes),
			"updated_at":        character.UpdatedAt,
		})
	}

	return result
}

func serializeCharacterDetail(character *models.Character) map[string]interface{} {
	equipment := make([]map[string]interface{}, 0, len(character.Equipment))
	for _, item := range character.Equipment {
		equipment = append(equipment, map[string]interface{}{
			"slot":              item.Slot,
			"inventory_item_id": item.InventoryItemID,
			"inventory_item":    serializeInventoryItem(item.InventoryItem),
		})
	}

	return map[string]interface{}{
		"id":               character.ID,
		"game_id":          character.GameID,
		"user_id":          character.UserID,
		"created_by_id":    character.CreatedByID,
		"name":             character.Name,
		"backstory":        character.Backstory,
		"portrait_id":      character.PortraitID,
		"inventory_width":  character.InventoryWidth,
		"inventory_height": character.InventoryHeight,
		"currency_gold":    character.CurrencyGold,
		"currency_silver":  character.CurrencySilver,
		"currency_copper":  character.CurrencyCopper,
		"owner": map[string]interface{}{
			"id":         character.User.ID,
			"username":   character.User.Username,
			"avatar_id":  character.User.AvatarID,
			"created_at": character.User.CreatedAt,
		},
		"base_attributes": map[string]int{
			"strength":     character.BaseStrength,
			"dexterity":    character.BaseDexterity,
			"constitution": character.BaseConstitution,
			"intelligence": character.BaseIntelligence,
			"wisdom":       character.BaseWisdom,
			"charisma":     character.BaseCharisma,
		},
		"custom_attributes": serializeCharacterCustomAttributes(character.CustomAttributes),
		"inventory":         serializeInventoryItems(character.Inventory),
		"equipment":         equipment,
		"created_at":        character.CreatedAt,
		"updated_at":        character.UpdatedAt,
	}
}

func serializeCharacterCustomAttributes(attributes []models.CharacterCustomAttribute) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(attributes))
	for _, attribute := range attributes {
		result = append(result, map[string]interface{}{
			"id":         attribute.ID,
			"name":       attribute.Name,
			"value":      attribute.Value,
			"sort_order": attribute.SortOrder,
		})
	}
	return result
}

func serializeInventoryItems(items []models.CharacterInventory) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, serializeInventoryItem(item))
	}
	return result
}

func serializeInventoryItem(item models.CharacterInventory) map[string]interface{} {
	return map[string]interface{}{
		"id":             item.ID,
		"character_id":   item.CharacterID,
		"item_id":        item.ItemID,
		"quantity":       item.Quantity,
		"durability":     item.Durability,
		"max_durability": item.MaxDurability,
		"enchantment":    item.Enchantment,
		"grid_x":         item.GridX,
		"grid_y":         item.GridY,
		"is_rotated":     item.IsRotated,
		"item":           serializeItem(item.Item),
		"updated_at":     item.UpdatedAt,
	}
}

func serializeItems(items []models.Item) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, serializeItem(item))
	}
	return result
}

func serializeItem(item models.Item) map[string]interface{} {
	types := make([]string, 0, len(item.Types))
	for _, itemType := range item.Types {
		types = append(types, itemType.TypeName)
	}

	requirements := make([]map[string]interface{}, 0, len(item.RequiredAttributes))
	for _, requirement := range item.RequiredAttributes {
		requirements = append(requirements, map[string]interface{}{
			"attribute_name": requirement.AttributeName,
			"min_value":      requirement.MinValue,
		})
	}

	modifiers := make([]map[string]interface{}, 0, len(item.AttributeModifiers))
	for _, modifier := range item.AttributeModifiers {
		modifiers = append(modifiers, map[string]interface{}{
			"attribute_name": modifier.AttributeName,
			"modifier_value": modifier.ModifierValue,
			"is_percentage":  modifier.IsPercentage,
		})
	}

	return map[string]interface{}{
		"id":                  item.ID,
		"game_id":             item.GameID,
		"created_by_id":       item.CreatedByID,
		"name":                item.Name,
		"description":         item.Description,
		"image_id":            item.ImageID,
		"rarity":              item.Rarity,
		"grid_width":          item.GridWidth,
		"grid_height":         item.GridHeight,
		"is_equippable":       item.IsEquippable,
		"equip_slot":          item.EquipSlot,
		"types":               types,
		"required_attributes": requirements,
		"attribute_modifiers": modifiers,
		"created_at":          item.CreatedAt,
		"updated_at":          item.UpdatedAt,
	}
}

func serializeChatMessages(messages []models.ChatMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, message := range messages {
		member := &models.GameMember{User: message.User}
		result = append(result, serializeChatMessage(&message, member))
	}
	return result
}

func serializeChatMessage(message *models.ChatMessage, member *models.GameMember) map[string]interface{} {
	username := "Unknown"
	var avatarID *string
	if member != nil {
		if member.User.Username != "" {
			username = member.User.Username
		}
		avatarID = member.User.AvatarID
	}

	return map[string]interface{}{
		"id":           message.ID,
		"game_id":      message.GameID,
		"user_id":      message.UserID,
		"message_type": message.MessageType,
		"content":      message.Content,
		"metadata":     message.Metadata,
		"created_at":   message.CreatedAt,
		"user": map[string]interface{}{
			"id":        message.UserID,
			"username":  username,
			"avatar_id": avatarID,
		},
	}
}

func findGameMember(game *models.Game, userID string) *models.GameMember {
	for i := range game.Members {
		if game.Members[i].UserID == userID {
			return &game.Members[i]
		}
	}
	return nil
}

func viewerCharacterLimit(isGM bool) int {
	if isGM {
		return -1
	}
	return nonGMCharacterLimit
}

func generateRandomCharacterName() string {
	firstParts := []string{
		"Ashen", "Iron", "Ivory", "Silent", "Scarlet", "Cinder", "Silver", "Storm", "Verdant", "Night",
	}
	secondParts := []string{
		"Rook", "Fox", "Blade", "Lantern", "Warden", "Wolf", "Hollow", "Raven", "Nomad", "Seer",
	}

	firstIndex := randomIndex(len(firstParts))
	secondIndex := randomIndex(len(secondParts))
	return fmt.Sprintf("%s %s", firstParts[firstIndex], secondParts[secondIndex])
}

func randomIndex(length int) int {
	if length <= 1 {
		return 0
	}

	value, err := crand.Int(crand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return int(time.Now().UnixNano() % int64(length))
	}

	return int(value.Int64())
}

func validateBaseAttribute(value int, label string) (int, error) {
	if value < 0 || value > 999 {
		return 0, fmt.Errorf("%s must be between 0 and 999", label)
	}

	return value, nil
}

func validateCurrencyAmount(value int, label string) (int, error) {
	if value < 0 || value > 999999999 {
		return 0, fmt.Errorf("%s must be between 0 and 999999999", label)
	}

	return value, nil
}
