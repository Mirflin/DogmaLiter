package game

import (
	crand "crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
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
const defaultGameItemPage = 1
const defaultGameItemPerPage = 18
const maxGameItemPerPage = 60

type updateCharacterBaseAttributesRequest struct {
	Strength     *int `json:"strength"`
	Dexterity    *int `json:"dexterity"`
	Constitution *int `json:"constitution"`
	Intelligence *int `json:"intelligence"`
	Wisdom       *int `json:"wisdom"`
	Charisma     *int `json:"charisma"`
}

type updateCharacterCustomAttributeRequest struct {
	ID        *string `json:"id"`
	Name      string  `json:"name"`
	Value     int     `json:"value"`
	SortOrder *int    `json:"sort_order"`
}

type createItemRequirementRequest struct {
	AttributeName string `json:"attribute_name"`
	MinValue      int    `json:"min_value"`
}

type createItemModifierRequest struct {
	AttributeName string `json:"attribute_name"`
	ModifierValue int    `json:"modifier_value"`
	IsPercentage  bool   `json:"is_percentage"`
}

type createItemRequest struct {
	Name               string                         `json:"name"`
	Description        string                         `json:"description"`
	Rarity             string                         `json:"rarity"`
	Category           string                         `json:"category"`
	Tags               []string                       `json:"tags"`
	GridWidth          *int                           `json:"grid_width"`
	GridHeight         *int                           `json:"grid_height"`
	EquipSlot          *string                        `json:"equip_slot"`
	RequiredAttributes []createItemRequirementRequest `json:"required_attributes"`
	AttributeModifiers []createItemModifierRequest    `json:"attribute_modifiers"`
}

type listGameItemsParams struct {
	Page     int
	PerPage  int
	Search   string
	Rarity   string
	Category string
	Slot     string
	Tag      string
	Sort     string
}

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

	itemTags, err := h.service.repo.ListGameItemTags(gameID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load item tags"})
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
		"item_tags":  serializeGameItemTags(itemTags),
		"messages":   serializeChatMessages(messages),
	})
}

func (h *Handler) ListItems(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can browse the compendium"})
		return
	}

	params, err := normalizeListGameItemsParams(r)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	items, totalItems, err := h.service.repo.ListGameItemsPage(gameID, params)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load items"})
		return
	}

	totalPages := 0
	if totalItems > 0 {
		totalPages = int((totalItems + int64(params.PerPage) - 1) / int64(params.PerPage))
	}

	respondJSON(w, 200, map[string]interface{}{
		"items": serializeItems(items),
		"pagination": map[string]interface{}{
			"page":        params.Page,
			"per_page":    params.PerPage,
			"total_items": totalItems,
			"total_pages": totalPages,
			"has_prev":    params.Page > 1,
			"has_next":    totalPages > 0 && params.Page < totalPages,
		},
	})
}

func (h *Handler) CreateCharacter(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	var req struct {
		Name        *string `json:"name"`
		Backstory   *string `json:"backstory"`
		OwnerUserID *string `json:"owner_user_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
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

	ownerUserID, err := resolveCharacterOwnerUserID(game, req.OwnerUserID, userID, isGM)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	name := generateRandomCharacterName()
	backstory := ""
	if isGM {
		name = "Untitled Character"
		if req.Name != nil {
			trimmedName := strings.TrimSpace(*req.Name)
			if trimmedName == "" {
				respondJSON(w, 400, map[string]string{"error": "Character name cannot be empty"})
				return
			}
			if len(trimmedName) > 100 {
				respondJSON(w, 400, map[string]string{"error": "Character name must be 100 characters or less"})
				return
			}
			name = trimmedName
		}

		if req.Backstory != nil {
			trimmedBackstory := strings.TrimSpace(*req.Backstory)
			if len(trimmedBackstory) > 5000 {
				respondJSON(w, 400, map[string]string{"error": "Backstory must be 5000 characters or less"})
				return
			}
			backstory = trimmedBackstory
		}
	}

	character := &models.Character{
		ID:               uuid.New().String(),
		GameID:           gameID,
		UserID:           ownerUserID,
		CreatedByID:      userID,
		Name:             name,
		Backstory:        backstory,
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

func (h *Handler) CreateItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can create items"})
		return
	}

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	plan, err := h.service.repo.GetUserPlan(game.OwnerID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load item limit"})
		return
	}

	itemCount, err := h.service.repo.CountGameItems(gameID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to check item limit"})
		return
	}
	if plan.MaxItemsPerGame != -1 && itemCount >= int64(plan.MaxItemsPerGame) {
		respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("Item limit reached (%d)", plan.MaxItemsPerGame)})
		return
	}

	item, err := normalizeCreateItemRequest(gameID, userID, req)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	tagNames, err := normalizeItemTagNames(req.Tags)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	if err := h.service.repo.CreateItem(item, tagNames); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to create item"})
		return
	}

	createdItem, err := h.service.repo.GetItemByID(gameID, item.ID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Item was created but could not be loaded"})
		return
	}

	respondJSON(w, 201, map[string]interface{}{
		"item": serializeItem(*createdItem),
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

	game, isGM, err := h.authorizeGameAccess(userID, gameID)
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
		Name             *string                                  `json:"name"`
		Backstory        *string                                  `json:"backstory"`
		CurrencyGold     *int                                     `json:"currency_gold"`
		CurrencySilver   *int                                     `json:"currency_silver"`
		CurrencyCopper   *int                                     `json:"currency_copper"`
		OwnerUserID      *string                                  `json:"owner_user_id"`
		InventoryWidth   *int                                     `json:"inventory_width"`
		InventoryHeight  *int                                     `json:"inventory_height"`
		BaseAttributes   *updateCharacterBaseAttributesRequest    `json:"base_attributes"`
		CustomAttributes *[]updateCharacterCustomAttributeRequest `json:"custom_attributes"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	hasChanges := false
	replaceCustomAttributes := false

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

	if req.OwnerUserID != nil || req.InventoryWidth != nil || req.InventoryHeight != nil || req.BaseAttributes != nil || req.CustomAttributes != nil {
		if !isGM {
			respondJSON(w, 403, map[string]string{"error": "Only the GM can edit advanced character settings"})
			return
		}
	}

	if req.OwnerUserID != nil {
		ownerUserID, err := resolveCharacterOwnerUserID(game, req.OwnerUserID, character.UserID, isGM)
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		character.UserID = ownerUserID
		hasChanges = true
	}

	if req.InventoryWidth != nil {
		value, err := validateInventoryDimension(*req.InventoryWidth, "Inventory width")
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		character.InventoryWidth = value
		hasChanges = true
	}

	if req.InventoryHeight != nil {
		value, err := validateInventoryDimension(*req.InventoryHeight, "Inventory height")
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
		character.InventoryHeight = value
		hasChanges = true
	}

	if req.BaseAttributes != nil {
		if req.BaseAttributes.Strength != nil {
			value, err := validateBaseAttribute(*req.BaseAttributes.Strength, "Strength")
			if err != nil {
				respondJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			character.BaseStrength = value
			hasChanges = true
		}

		if req.BaseAttributes.Dexterity != nil {
			value, err := validateBaseAttribute(*req.BaseAttributes.Dexterity, "Dexterity")
			if err != nil {
				respondJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			character.BaseDexterity = value
			hasChanges = true
		}

		if req.BaseAttributes.Constitution != nil {
			value, err := validateBaseAttribute(*req.BaseAttributes.Constitution, "Constitution")
			if err != nil {
				respondJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			character.BaseConstitution = value
			hasChanges = true
		}

		if req.BaseAttributes.Intelligence != nil {
			value, err := validateBaseAttribute(*req.BaseAttributes.Intelligence, "Intelligence")
			if err != nil {
				respondJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			character.BaseIntelligence = value
			hasChanges = true
		}

		if req.BaseAttributes.Wisdom != nil {
			value, err := validateBaseAttribute(*req.BaseAttributes.Wisdom, "Wisdom")
			if err != nil {
				respondJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			character.BaseWisdom = value
			hasChanges = true
		}

		if req.BaseAttributes.Charisma != nil {
			value, err := validateBaseAttribute(*req.BaseAttributes.Charisma, "Charisma")
			if err != nil {
				respondJSON(w, 400, map[string]string{"error": err.Error()})
				return
			}
			character.BaseCharisma = value
			hasChanges = true
		}
	}

	if req.CustomAttributes != nil {
		attributes, err := normalizeCustomAttributes(*req.CustomAttributes)
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}

		for index := range attributes {
			attributes[index].CharacterID = character.ID
		}

		character.CustomAttributes = attributes
		replaceCustomAttributes = true
		hasChanges = true
	}

	if req.InventoryWidth != nil || req.InventoryHeight != nil {
		if err := validateInventoryResize(character); err != nil {
			respondJSON(w, 400, map[string]string{"error": err.Error()})
			return
		}
	}

	if hasChanges {
		if err := h.service.repo.UpdateCharacter(character, replaceCustomAttributes); err != nil {
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
	types := make([]string, 0, len(item.Types)+1)
	if item.Category != "" {
		types = append(types, item.Category)
	}
	for _, itemType := range item.Types {
		typeName := strings.TrimSpace(itemType.TypeName)
		if typeName == "" {
			continue
		}

		if strings.EqualFold(typeName, item.Category) {
			continue
		}

		types = append(types, typeName)
	}

	normalizedEquipSlot := normalizeItemEquipSlotValue(item.EquipSlot)

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

	tags := make([]string, 0, len(item.Tags))
	for _, tag := range item.Tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		tags = append(tags, name)
	}

	return map[string]interface{}{
		"id":                  item.ID,
		"game_id":             item.GameID,
		"created_by_id":       item.CreatedByID,
		"name":                item.Name,
		"description":         item.Description,
		"image_id":            item.ImageID,
		"rarity":              item.Rarity,
		"category":            item.Category,
		"grid_width":          item.GridWidth,
		"grid_height":         item.GridHeight,
		"is_equippable":       normalizedEquipSlot != nil,
		"equip_slot":          normalizedEquipSlot,
		"tags":                tags,
		"types":               types,
		"required_attributes": requirements,
		"attribute_modifiers": modifiers,
		"created_at":          item.CreatedAt,
		"updated_at":          item.UpdatedAt,
	}
}

func serializeGameItemTags(tags []models.GameItemTag) []string {
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		result = append(result, name)
	}
	return result
}

func serializeChatMessages(messages []models.ChatMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(messages))
	for _, message := range messages {
		member := &models.GameMember{User: message.User}
		result = append(result, serializeChatMessage(&message, member))
	}
	return result
}

func normalizeListGameItemsParams(r *http.Request) (listGameItemsParams, error) {
	page, err := normalizePositiveQueryInt(r.URL.Query().Get("page"), defaultGameItemPage)
	if err != nil {
		return listGameItemsParams{}, fmt.Errorf("Page must be a positive integer")
	}

	perPage, err := normalizePositiveQueryInt(r.URL.Query().Get("per_page"), defaultGameItemPerPage)
	if err != nil {
		return listGameItemsParams{}, fmt.Errorf("Per-page value must be a positive integer")
	}
	if perPage > maxGameItemPerPage {
		perPage = maxGameItemPerPage
	}

	search := strings.Join(strings.Fields(strings.TrimSpace(r.URL.Query().Get("search"))), " ")
	if len(search) > 200 {
		return listGameItemsParams{}, fmt.Errorf("Search query must be 200 characters or less")
	}

	rarity := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("rarity")))
	if rarity == "all" {
		rarity = ""
	}
	if rarity != "" {
		normalizedRarity, err := normalizeItemRarity(rarity)
		if err != nil {
			return listGameItemsParams{}, err
		}
		rarity = normalizedRarity
	}

	category := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("category")))
	if category == "all" {
		category = ""
	}
	if category != "" {
		normalizedCategory, err := normalizeItemCategory(category)
		if err != nil {
			return listGameItemsParams{}, err
		}
		category = normalizedCategory
	}

	slot := strings.TrimSpace(strings.ToLower(r.URL.Query().Get("slot")))
	if slot == "all" {
		slot = ""
	}
	if slot != "" {
		normalizedSlot, err := normalizeItemEquipSlotValueForCreate(&slot)
		if err != nil {
			return listGameItemsParams{}, err
		}
		if normalizedSlot == nil {
			slot = ""
		} else {
			slot = *normalizedSlot
		}
	}

	tagValues, err := normalizeItemTagNames([]string{r.URL.Query().Get("tag")})
	if err != nil {
		return listGameItemsParams{}, err
	}
	tag := ""
	if len(tagValues) > 0 {
		tag = tagValues[0]
	}

	return listGameItemsParams{
		Page:     page,
		PerPage:  perPage,
		Search:   search,
		Rarity:   rarity,
		Category: category,
		Slot:     slot,
		Tag:      tag,
		Sort:     normalizeItemListSort(r.URL.Query().Get("sort")),
	}, nil
}

func normalizePositiveQueryInt(rawValue string, fallback int) (int, error) {
	trimmedValue := strings.TrimSpace(rawValue)
	if trimmedValue == "" {
		return fallback, nil
	}

	parsedValue, err := strconv.Atoi(trimmedValue)
	if err != nil || parsedValue < 1 {
		return 0, fmt.Errorf("invalid positive integer")
	}

	return parsedValue, nil
}

func normalizeItemListSort(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "name-asc", "name-desc", "rarity", "size":
		return strings.TrimSpace(strings.ToLower(value))
	default:
		return "recent"
	}
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

func resolveCharacterOwnerUserID(game *models.Game, requestedOwnerID *string, fallbackUserID string, isGM bool) (string, error) {
	ownerUserID := fallbackUserID
	if requestedOwnerID == nil {
		return ownerUserID, nil
	}

	if !isGM {
		return "", fmt.Errorf("Only the GM can reassign characters")
	}

	trimmedUserID := strings.TrimSpace(*requestedOwnerID)
	if trimmedUserID == "" {
		return "", fmt.Errorf("Character owner is required")
	}

	member := findGameMember(game, trimmedUserID)
	if member == nil {
		return "", fmt.Errorf("Selected character owner is not a member of this game")
	}

	return member.UserID, nil
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

func validateInventoryDimension(value int, label string) (int, error) {
	if value < 1 || value > 20 {
		return 0, fmt.Errorf("%s must be between 1 and 20", label)
	}

	return value, nil
}

func validateInventoryResize(character *models.Character) error {
	for _, entry := range character.Inventory {
		itemWidth := entry.Item.GridWidth
		itemHeight := entry.Item.GridHeight
		if itemWidth < 1 {
			itemWidth = 1
		}
		if itemHeight < 1 {
			itemHeight = 1
		}
		if entry.IsRotated {
			itemWidth, itemHeight = itemHeight, itemWidth
		}

		if entry.GridX < 0 || entry.GridY < 0 || entry.GridX+itemWidth > character.InventoryWidth || entry.GridY+itemHeight > character.InventoryHeight {
			itemName := strings.TrimSpace(entry.Item.Name)
			if itemName == "" {
				itemName = "An item"
			}
			return fmt.Errorf("%s no longer fits inside a %dx%d inventory", itemName, character.InventoryWidth, character.InventoryHeight)
		}
	}

	return nil
}

func normalizeCustomAttributes(input []updateCharacterCustomAttributeRequest) ([]models.CharacterCustomAttribute, error) {
	if len(input) > 50 {
		return nil, fmt.Errorf("Custom attributes must contain 50 entries or fewer")
	}

	attributes := make([]models.CharacterCustomAttribute, 0, len(input))
	seenNames := make(map[string]struct{}, len(input))

	for index, attribute := range input {
		name := strings.TrimSpace(attribute.Name)
		if name == "" {
			return nil, fmt.Errorf("Custom attribute name cannot be empty")
		}
		if len(name) > 100 {
			return nil, fmt.Errorf("Custom attribute name must be 100 characters or less")
		}

		normalizedKey := strings.ToLower(name)
		if _, exists := seenNames[normalizedKey]; exists {
			return nil, fmt.Errorf("Custom attribute names must be unique")
		}
		seenNames[normalizedKey] = struct{}{}

		value, err := validateCustomAttributeValue(attribute.Value)
		if err != nil {
			return nil, err
		}

		sortOrder := index
		if attribute.SortOrder != nil {
			sortOrder = *attribute.SortOrder
		}

		attributeID := uuid.New().String()
		if attribute.ID != nil && strings.TrimSpace(*attribute.ID) != "" {
			attributeID = strings.TrimSpace(*attribute.ID)
		}

		attributes = append(attributes, models.CharacterCustomAttribute{
			ID:        attributeID,
			Name:      name,
			Value:     value,
			SortOrder: sortOrder,
		})
	}

	return attributes, nil
}

func validateCustomAttributeValue(value int) (int, error) {
	if value < -999999999 || value > 999999999 {
		return 0, fmt.Errorf("Custom attribute values must be between -999999999 and 999999999")
	}

	return value, nil
}

func validateCurrencyAmount(value int, label string) (int, error) {
	if value < 0 || value > 999999999 {
		return 0, fmt.Errorf("%s must be between 0 and 999999999", label)
	}

	return value, nil
}

func normalizeCreateItemRequest(gameID, userID string, req createItemRequest) (*models.Item, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("Item name cannot be empty")
	}
	if len(name) > 200 {
		return nil, fmt.Errorf("Item name must be 200 characters or less")
	}

	description := strings.TrimSpace(req.Description)
	if len(description) > 5000 {
		return nil, fmt.Errorf("Item description must be 5000 characters or less")
	}

	rarity, err := normalizeItemRarity(req.Rarity)
	if err != nil {
		return nil, err
	}

	category, err := normalizeItemCategory(req.Category)
	if err != nil {
		return nil, err
	}

	equipSlot, err := normalizeItemEquipSlotValueForCreate(req.EquipSlot)
	if err != nil {
		return nil, err
	}

	gridWidth := 1
	if req.GridWidth != nil {
		gridWidth = *req.GridWidth
	}
	if gridWidth < 1 || gridWidth > 20 {
		return nil, fmt.Errorf("Item width must be between 1 and 20")
	}

	gridHeight := 1
	if req.GridHeight != nil {
		gridHeight = *req.GridHeight
	}
	if gridHeight < 1 || gridHeight > 20 {
		return nil, fmt.Errorf("Item height must be between 1 and 20")
	}

	requirements, err := normalizeItemRequirements(req.RequiredAttributes)
	if err != nil {
		return nil, err
	}

	modifiers, err := normalizeItemModifiers(req.AttributeModifiers)
	if err != nil {
		return nil, err
	}

	itemID := uuid.New().String()
	for index := range requirements {
		requirements[index].ID = uuid.New().String()
		requirements[index].ItemID = itemID
	}
	for index := range modifiers {
		modifiers[index].ID = uuid.New().String()
		modifiers[index].ItemID = itemID
	}

	return &models.Item{
		ID:                 itemID,
		GameID:             gameID,
		CreatedByID:        userID,
		Name:               name,
		Description:        description,
		Rarity:             rarity,
		Category:           category,
		GridWidth:          gridWidth,
		GridHeight:         gridHeight,
		EquipSlot:          equipSlot,
		RequiredAttributes: requirements,
		AttributeModifiers: modifiers,
	}, nil
}

func normalizeItemTagNames(input []string) ([]string, error) {
	if len(input) > 20 {
		return nil, fmt.Errorf("Item tags must contain 20 entries or fewer")
	}

	result := make([]string, 0, len(input))
	seen := make(map[string]struct{}, len(input))

	for _, rawValue := range input {
		name := strings.Join(strings.Fields(strings.TrimSpace(rawValue)), " ")
		if name == "" {
			continue
		}
		if len(name) > 60 {
			return nil, fmt.Errorf("Item tag names must be 60 characters or less")
		}

		lookupKey := strings.ToLower(name)
		if _, exists := seen[lookupKey]; exists {
			continue
		}
		seen[lookupKey] = struct{}{}
		result = append(result, name)
	}

	return result, nil
}

func normalizeItemRarity(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return models.ItemRarityCommon, nil
	}
	if normalized == "artifact" {
		return models.ItemRarityUnique, nil
	}

	for _, allowed := range models.ValidItemRarities {
		if normalized == allowed {
			return normalized, nil
		}
	}

	return "", fmt.Errorf("Unsupported item rarity")
}

func normalizeItemCategory(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return models.ItemCategoryOther, nil
	}

	for _, allowed := range models.ValidItemCategories {
		if normalized == allowed {
			return normalized, nil
		}
	}

	return "", fmt.Errorf("Unsupported item category")
}

func normalizeItemEquipSlotValue(slot *string) *string {
	if slot == nil {
		return nil
	}

	normalized, err := normalizeItemEquipSlotValueForCreate(slot)
	if err != nil {
		return nil
	}

	return normalized
}

func normalizeItemEquipSlotValueForCreate(slot *string) (*string, error) {
	if slot == nil {
		return nil, nil
	}

	normalized := strings.ToLower(strings.TrimSpace(*slot))
	if normalized == "" {
		return nil, nil
	}
	if normalized == "ring_1" || normalized == "ring_2" {
		normalized = models.ItemEquipSlotRing
	}

	for _, allowed := range models.ValidItemEquipSlots {
		if normalized == allowed {
			return &normalized, nil
		}
	}

	return nil, fmt.Errorf("Unsupported item equip slot")
}

func normalizeItemRequirements(input []createItemRequirementRequest) ([]models.ItemRequiredAttribute, error) {
	if len(input) > 50 {
		return nil, fmt.Errorf("Item requirements must contain 50 entries or fewer")
	}

	requirements := make([]models.ItemRequiredAttribute, 0, len(input))
	seenNames := make(map[string]struct{}, len(input))

	for _, entry := range input {
		attributeName := strings.ToLower(strings.TrimSpace(entry.AttributeName))
		if attributeName == "" {
			return nil, fmt.Errorf("Requirement attribute name cannot be empty")
		}
		if len(attributeName) > 50 {
			return nil, fmt.Errorf("Requirement attribute name must be 50 characters or less")
		}
		if _, exists := seenNames[attributeName]; exists {
			return nil, fmt.Errorf("Requirement attributes must be unique")
		}
		seenNames[attributeName] = struct{}{}

		if entry.MinValue < 0 || entry.MinValue > 999999999 {
			return nil, fmt.Errorf("Requirement values must be between 0 and 999999999")
		}

		requirements = append(requirements, models.ItemRequiredAttribute{
			AttributeName: attributeName,
			MinValue:      entry.MinValue,
		})
	}

	return requirements, nil
}

func normalizeItemModifiers(input []createItemModifierRequest) ([]models.ItemAttributeModifier, error) {
	if len(input) > 50 {
		return nil, fmt.Errorf("Item modifiers must contain 50 entries or fewer")
	}

	modifiers := make([]models.ItemAttributeModifier, 0, len(input))
	seenNames := make(map[string]struct{}, len(input))

	for _, entry := range input {
		attributeName := strings.ToLower(strings.TrimSpace(entry.AttributeName))
		if attributeName == "" {
			return nil, fmt.Errorf("Modifier attribute name cannot be empty")
		}
		if len(attributeName) > 50 {
			return nil, fmt.Errorf("Modifier attribute name must be 50 characters or less")
		}
		if _, exists := seenNames[attributeName]; exists {
			return nil, fmt.Errorf("Modifier attributes must be unique")
		}
		seenNames[attributeName] = struct{}{}

		if entry.ModifierValue < -999999999 || entry.ModifierValue > 999999999 {
			return nil, fmt.Errorf("Modifier values must be between -999999999 and 999999999")
		}

		modifiers = append(modifiers, models.ItemAttributeModifier{
			AttributeName: attributeName,
			ModifierValue: entry.ModifierValue,
			IsPercentage:  entry.IsPercentage,
		})
	}

	return modifiers, nil
}
