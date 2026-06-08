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
const gameCharacterLimit = 5
const gameChatMessageLimit = 40
const defaultGameItemPage = 1
const defaultGameItemPerPage = 18
const maxGameItemPerPage = 60
const maxItemNameLength = 20
const maxItemDescriptionLength = 100
const maxItemDescriptionLineBreaks = 1

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

	gameCharacterCount, err := h.service.repo.CountGameCharacters(gameID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to check character limit"})
		return
	}
	if gameCharacterCount >= gameCharacterLimit {
		respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("This game has reached its character limit (%d)", gameCharacterLimit)})
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
	h.logActivity(gameID, userID, character.Name, "Created character", "")

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

func (h *Handler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	itemID := chi.URLParam(r, "itemID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can update items"})
		return
	}

	existingItem, err := h.service.repo.GetItemByID(gameID, itemID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": fmt.Sprintf("Item %s not found", itemID)})
		return
	}

	var req createItemRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	item, err := normalizeItemRequest(itemID, gameID, existingItem.CreatedByID, req)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	tagNames, err := normalizeItemTagNames(req.Tags)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	if err := h.service.repo.UpdateItem(item, tagNames); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to update item"})
		return
	}

	updatedItem, err := h.service.repo.GetItemByID(gameID, itemID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Item was updated but could not be loaded"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"item": serializeItem(*updatedItem),
	})
}

func (h *Handler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	itemID := chi.URLParam(r, "itemID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can delete items"})
		return
	}

	item, err := h.service.repo.GetItemByID(gameID, itemID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": fmt.Sprintf("Item %s not found", itemID)})
		return
	}

	if err := h.service.DeleteItem(item); err != nil {
		respondJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"deleted_item_id": itemID,
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
		h.logActivity(gameID, userID, character.Name, "Updated character", "")

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

func (h *Handler) GiveInventoryItems(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	characterID := chi.URLParam(r, "characterID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can deliver items to characters"})
		return
	}

	character, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": fmt.Sprintf("Character %s not found", characterID)})
		return
	}

	var req struct {
		Items []struct {
			ItemID        string `json:"item_id"`
			Quantity      *int   `json:"quantity"`
			Durability    *int   `json:"durability"`
			MaxDurability *int   `json:"max_durability"`
			Enchantment   *int   `json:"enchantment"`
		} `json:"items"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}
	if len(req.Items) == 0 {
		respondJSON(w, 400, map[string]string{"error": "Select at least one item to deliver"})
		return
	}
	if len(req.Items) > 50 {
		respondJSON(w, 400, map[string]string{"error": "You can deliver up to 50 items at once"})
		return
	}

	width := character.InventoryWidth
	height := character.InventoryHeight
	occupied := make([][]bool, height)
	for y := range occupied {
		occupied[y] = make([]bool, width)
	}
	markInventoryOccupancy(occupied, character.Inventory)

	entries := make([]models.CharacterInventory, 0, len(req.Items))
	unplaced := make([]string, 0)

	for _, line := range req.Items {
		item, err := h.service.repo.GetItemByID(gameID, line.ItemID)
		if err != nil {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("Item %s is not part of this compendium", line.ItemID)})
			return
		}

		quantity := 1
		if line.Quantity != nil {
			quantity = *line.Quantity
		}
		if quantity < 1 || quantity > 9999 {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("%s: quantity must be between 1 and 9999", item.Name)})
			return
		}

		enchantment := 0
		if line.Enchantment != nil {
			enchantment = *line.Enchantment
		}
		if enchantment < -999 || enchantment > 999 {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("%s: enchantment must be between -999 and 999", item.Name)})
			return
		}

		var durability, maxDurability *int
		if line.MaxDurability != nil {
			value := *line.MaxDurability
			if value < 1 || value > 1000000 {
				respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("%s: max durability must be between 1 and 1,000,000", item.Name)})
				return
			}
			maxDurability = &value
		}
		if line.Durability != nil {
			value := *line.Durability
			if value < 0 || value > 1000000 {
				respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("%s: durability must be between 0 and 1,000,000", item.Name)})
				return
			}
			if maxDurability != nil && value > *maxDurability {
				respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("%s: durability cannot exceed max durability", item.Name)})
				return
			}
			durability = &value
		}

		gridX, gridY, placed := findFreeInventorySlot(occupied, width, height, item.GridWidth, item.GridHeight)
		if !placed {
			unplaced = append(unplaced, item.Name)
			continue
		}
		markCells(occupied, gridX, gridY, item.GridWidth, item.GridHeight, true)

		entries = append(entries, models.CharacterInventory{
			ID:            uuid.New().String(),
			CharacterID:   character.ID,
			ItemID:        item.ID,
			Quantity:      quantity,
			Durability:    durability,
			MaxDurability: maxDurability,
			Enchantment:   enchantment,
			GridX:         gridX,
			GridY:         gridY,
			IsRotated:     false,
		})
	}

	if len(unplaced) > 0 {
		respondJSON(w, 400, map[string]string{
			"error": fmt.Sprintf("%s has no free inventory space for: %s", character.Name, strings.Join(unplaced, ", ")),
		})
		return
	}

	if err := h.service.repo.AddCharacterInventoryItems(entries); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to deliver items"})
		return
	}
	h.logActivity(gameID, userID, character.Name, "Delivered items", fmt.Sprintf("%d item(s)", len(entries)))

	updated, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Items were delivered but the character could not be reloaded"})
		return
	}

	respondJSON(w, 201, map[string]interface{}{
		"character": serializeCharacterDetail(updated),
		"delivered": len(entries),
	})
}

func (h *Handler) UpdateInventoryLayout(w http.ResponseWriter, r *http.Request) {
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
		Inventory []struct {
			ID        string `json:"id"`
			GridX     int    `json:"grid_x"`
			GridY     int    `json:"grid_y"`
			IsRotated bool   `json:"is_rotated"`
		} `json:"inventory"`
		Equipment []struct {
			Slot            string `json:"slot"`
			InventoryItemID string `json:"inventory_item_id"`
		} `json:"equipment"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	inventoryByID := make(map[string]models.CharacterInventory, len(character.Inventory))
	for _, entry := range character.Inventory {
		inventoryByID[entry.ID] = entry
	}

	positions := make([]InventoryPositionUpdate, 0, len(req.Inventory))
	for _, line := range req.Inventory {
		entry, ok := inventoryByID[line.ID]
		if !ok {
			respondJSON(w, 400, map[string]string{"error": "Inventory item does not belong to this character"})
			return
		}

		width := entry.Item.GridWidth
		height := entry.Item.GridHeight
		if line.IsRotated {
			width, height = height, width
		}
		if width < 1 {
			width = 1
		}
		if height < 1 {
			height = 1
		}
		if line.GridX < 0 || line.GridY < 0 || line.GridX+width > character.InventoryWidth || line.GridY+height > character.InventoryHeight {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("%s does not fit at the requested position", entry.Item.Name)})
			return
		}

		positions = append(positions, InventoryPositionUpdate{
			ID:        line.ID,
			GridX:     line.GridX,
			GridY:     line.GridY,
			IsRotated: line.IsRotated,
		})
	}

	validSlots := make(map[string]bool, len(models.ValidSlots))
	for _, slot := range models.ValidSlots {
		validSlots[slot] = true
	}

	seenSlots := make(map[string]bool)
	seenItems := make(map[string]bool)
	equipment := make([]models.CharacterEquipment, 0, len(req.Equipment))
	for _, line := range req.Equipment {
		if !validSlots[line.Slot] {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("Unknown equipment slot %s", line.Slot)})
			return
		}
		entry, ok := inventoryByID[line.InventoryItemID]
		if !ok {
			respondJSON(w, 400, map[string]string{"error": "Equipped item does not belong to this character"})
			return
		}
		if !equipSlotMatches(entry.Item.EquipSlot, line.Slot) {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("%s cannot be equipped in the %s slot", entry.Item.Name, line.Slot)})
			return
		}
		if seenSlots[line.Slot] {
			respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("Slot %s was assigned more than once", line.Slot)})
			return
		}
		if seenItems[line.InventoryItemID] {
			respondJSON(w, 400, map[string]string{"error": "An item cannot be equipped in more than one slot"})
			return
		}
		seenSlots[line.Slot] = true
		seenItems[line.InventoryItemID] = true

		equipment = append(equipment, models.CharacterEquipment{
			CharacterID:     characterID,
			Slot:            line.Slot,
			InventoryItemID: line.InventoryItemID,
		})
	}

	if err := h.service.repo.UpdateInventoryLayout(characterID, positions, equipment); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to save inventory layout"})
		return
	}
	{
		nameByID := make(map[string]string, len(character.Inventory))
		oldGrid := make(map[string][2]int, len(character.Inventory))
		for _, inv := range character.Inventory {
			nameByID[inv.ID] = inv.Item.Name
			oldGrid[inv.ID] = [2]int{inv.GridX, inv.GridY}
		}
		oldEquip := make(map[string]string, len(character.Equipment))
		for _, eq := range character.Equipment {
			oldEquip[eq.InventoryItemID] = eq.Slot
		}
		newEquip := make(map[string]string, len(equipment))
		for _, eq := range equipment {
			newEquip[eq.InventoryItemID] = eq.Slot
		}

		itemLabel := func(id string) string {
			if name := nameByID[id]; name != "" {
				return name
			}
			return "an item"
		}

		var changes []string
		for itemID, slot := range newEquip {
			if oldEquip[itemID] != slot {
				changes = append(changes, fmt.Sprintf("equipped %s (%s)", itemLabel(itemID), slot))
			}
		}
		for itemID := range oldEquip {
			if _, stillEquipped := newEquip[itemID]; !stillEquipped {
				changes = append(changes, fmt.Sprintf("unequipped %s", itemLabel(itemID)))
			}
		}
		for _, position := range positions {
			if _, wasEquipped := oldEquip[position.ID]; wasEquipped {
				continue
			}
			if _, nowEquipped := newEquip[position.ID]; nowEquipped {
				continue
			}
			if old, ok := oldGrid[position.ID]; ok && (old[0] != position.GridX || old[1] != position.GridY) {
				changes = append(changes, fmt.Sprintf("moved %s", itemLabel(position.ID)))
			}
		}

		if len(changes) > 0 {
			extra := 0
			if len(changes) > 6 {
				extra = len(changes) - 6
				changes = changes[:6]
			}
			details := strings.Join(changes, ", ")
			if extra > 0 {
				details += fmt.Sprintf(" +%d more", extra)
			}
			h.logActivity(gameID, userID, character.Name, "Inventory", details)
		}
	}

	updated, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Inventory layout was saved but the character could not be reloaded"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"character": serializeCharacterDetail(updated),
	})
}

func equipSlotMatches(itemEquip *string, slot string) bool {
	if itemEquip == nil {
		return false
	}
	value := *itemEquip
	if value == models.ItemEquipSlotRing {
		return slot == models.SlotRing1 || slot == models.SlotRing2
	}
	return value == slot
}

func (h *Handler) UpdateInventoryItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	characterID := chi.URLParam(r, "characterID")
	inventoryItemID := chi.URLParam(r, "inventoryItemID")

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

	var target *models.CharacterInventory
	for i := range character.Inventory {
		if character.Inventory[i].ID == inventoryItemID {
			target = &character.Inventory[i]
			break
		}
	}
	if target == nil {
		respondJSON(w, 404, map[string]string{"error": "Inventory item not found"})
		return
	}

	var req struct {
		Durability    *int `json:"durability"`
		MaxDurability *int `json:"max_durability"`
		Enchantment   *int `json:"enchantment"`
		Quantity      *int `json:"quantity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}

	updates := map[string]interface{}{}
	effectiveMax := target.MaxDurability

	if req.MaxDurability != nil {
		value := *req.MaxDurability
		if value < 1 || value > 1000000 {
			respondJSON(w, 400, map[string]string{"error": "Max durability must be between 1 and 1,000,000"})
			return
		}
		updates["max_durability"] = value
		effectiveMax = &value
	}
	if req.Durability != nil {
		value := *req.Durability
		if value < 0 || value > 1000000 {
			respondJSON(w, 400, map[string]string{"error": "Durability must be between 0 and 1,000,000"})
			return
		}
		if effectiveMax != nil && value > *effectiveMax {
			respondJSON(w, 400, map[string]string{"error": "Durability cannot exceed max durability"})
			return
		}
		updates["durability"] = value
	}
	if req.Enchantment != nil {
		value := *req.Enchantment
		if value < -999 || value > 999 {
			respondJSON(w, 400, map[string]string{"error": "Enchantment must be between -999 and 999"})
			return
		}
		updates["enchantment"] = value
	}
	if req.Quantity != nil {
		value := *req.Quantity
		if value < 1 || value > 9999 {
			respondJSON(w, 400, map[string]string{"error": "Quantity must be between 1 and 9999"})
			return
		}
		updates["quantity"] = value
	}

	if len(updates) == 0 {
		respondJSON(w, 400, map[string]string{"error": "No changes were provided"})
		return
	}
	updates["updated_at"] = time.Now()

	if _, err := h.service.repo.UpdateInventoryEntry(characterID, inventoryItemID, updates); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to update inventory item"})
		return
	}
	h.logActivity(gameID, userID, character.Name, "Updated item", target.Item.Name)

	updated, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Item was updated but the character could not be reloaded"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"character": serializeCharacterDetail(updated),
	})
}

func (h *Handler) SplitInventoryItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	characterID := chi.URLParam(r, "characterID")
	inventoryItemID := chi.URLParam(r, "inventoryItemID")

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

	var target *models.CharacterInventory
	for i := range character.Inventory {
		if character.Inventory[i].ID == inventoryItemID {
			target = &character.Inventory[i]
			break
		}
	}
	if target == nil {
		respondJSON(w, 404, map[string]string{"error": "Inventory item not found"})
		return
	}
	if target.Quantity <= 1 {
		respondJSON(w, 400, map[string]string{"error": "This item cannot be unstacked"})
		return
	}

	occupied := make([][]bool, character.InventoryHeight)
	for y := range occupied {
		occupied[y] = make([]bool, character.InventoryWidth)
	}
	markInventoryOccupancy(occupied, character.Inventory)

	width := target.Item.GridWidth
	height := target.Item.GridHeight
	if target.IsRotated {
		width, height = height, width
	}

	gridX, gridY, placed := findFreeInventorySlot(occupied, character.InventoryWidth, character.InventoryHeight, width, height)
	if !placed {
		respondJSON(w, 400, map[string]string{"error": "No free inventory space to unstack this item"})
		return
	}

	newEntry := models.CharacterInventory{
		ID:            uuid.New().String(),
		CharacterID:   character.ID,
		ItemID:        target.ItemID,
		Quantity:      1,
		Durability:    copyIntPointer(target.Durability),
		MaxDurability: copyIntPointer(target.MaxDurability),
		Enchantment:   target.Enchantment,
		GridX:         gridX,
		GridY:         gridY,
		IsRotated:     target.IsRotated,
	}

	if err := h.service.repo.SplitInventoryItem(target.ID, target.Quantity-1, newEntry); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to unstack item"})
		return
	}
	h.logActivity(gameID, userID, character.Name, "Unstacked item", target.Item.Name)

	updated, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Item was unstacked but the character could not be reloaded"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"character": serializeCharacterDetail(updated),
	})
}

func copyIntPointer(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func (h *Handler) DeleteInventoryItem(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	characterID := chi.URLParam(r, "characterID")
	inventoryItemID := chi.URLParam(r, "inventoryItemID")

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

	removedItemName := ""
	for i := range character.Inventory {
		if character.Inventory[i].ID == inventoryItemID {
			removedItemName = character.Inventory[i].Item.Name
			break
		}
	}

	affected, err := h.service.repo.DeleteInventoryEntry(characterID, inventoryItemID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to delete inventory item"})
		return
	}
	if affected == 0 {
		respondJSON(w, 404, map[string]string{"error": "Inventory item not found"})
		return
	}
	h.logActivity(gameID, userID, character.Name, "Removed item", removedItemName)

	updated, err := h.service.repo.GetCharacterByID(gameID, characterID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Item was removed but the character could not be reloaded"})
		return
	}

	respondJSON(w, 200, map[string]interface{}{
		"character": serializeCharacterDetail(updated),
	})
}

func (h *Handler) DeleteCharacter(w http.ResponseWriter, r *http.Request) {
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
		respondJSON(w, 403, map[string]string{"error": "You do not have access to delete this character"})
		return
	}

	if err := h.service.repo.DeleteCharacter(gameID, characterID); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to delete character"})
		return
	}
	h.logActivity(gameID, userID, character.Name, "Deleted character", "")

	respondJSON(w, 200, map[string]interface{}{
		"success":      true,
		"character_id": characterID,
	})
}

func (h *Handler) logActivity(gameID, userID, characterName, action, details string) {
	_ = h.service.repo.CreateActivity(&models.ActivityLog{
		ID:            uuid.New().String(),
		GameID:        gameID,
		UserID:        userID,
		CharacterName: characterName,
		Action:        action,
		Details:       details,
		CreatedAt:     time.Now(),
	})
}

func clearCutoff(r *http.Request) *time.Time {
	hours, _ := strconv.Atoi(r.URL.Query().Get("older_than_hours"))
	if hours <= 0 {
		return nil
	}
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	return &cutoff
}

func (h *Handler) ClearChatMessages(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can clear chat history"})
		return
	}

	deleted, err := h.service.repo.DeleteChatMessages(gameID, clearCutoff(r))
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to clear chat history"})
		return
	}
	respondJSON(w, 200, map[string]interface{}{"success": true, "deleted": deleted})
}

func (h *Handler) ClearActivity(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can clear activity"})
		return
	}

	deleted, err := h.service.repo.DeleteActivity(gameID, clearCutoff(r))
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to clear activity"})
		return
	}
	respondJSON(w, 200, map[string]interface{}{"success": true, "deleted": deleted})
}

func (h *Handler) GetActivity(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	_, isGM, err := h.authorizeGameAccess(userID, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can view activity"})
		return
	}

	logs, err := h.service.repo.ListActivity(gameID, 200)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load activity"})
		return
	}

	result := make([]map[string]interface{}, 0, len(logs))
	for _, entry := range logs {
		result = append(result, map[string]interface{}{
			"id":             entry.ID,
			"action":         entry.Action,
			"details":        entry.Details,
			"character_name": entry.CharacterName,
			"created_at":     entry.CreatedAt,
			"user": map[string]interface{}{
				"id":        entry.User.ID,
				"username":  entry.User.Username,
				"avatar_id": entry.User.AvatarID,
			},
		})
	}

	respondJSON(w, 200, map[string]interface{}{"activity": result})
}

func markInventoryOccupancy(occupied [][]bool, items []models.CharacterInventory) {
	for _, entry := range items {
		width := entry.Item.GridWidth
		height := entry.Item.GridHeight
		if width < 1 {
			width = 1
		}
		if height < 1 {
			height = 1
		}
		if entry.IsRotated {
			width, height = height, width
		}
		markCells(occupied, entry.GridX, entry.GridY, width, height, true)
	}
}

func markCells(occupied [][]bool, startX, startY, width, height int, value bool) {
	for y := startY; y < startY+height; y++ {
		if y < 0 || y >= len(occupied) {
			continue
		}
		for x := startX; x < startX+width; x++ {
			if x < 0 || x >= len(occupied[y]) {
				continue
			}
			occupied[y][x] = value
		}
	}
}

func findFreeInventorySlot(occupied [][]bool, gridWidth, gridHeight, itemWidth, itemHeight int) (int, int, bool) {
	if itemWidth < 1 {
		itemWidth = 1
	}
	if itemHeight < 1 {
		itemHeight = 1
	}
	if itemWidth > gridWidth || itemHeight > gridHeight {
		return 0, 0, false
	}
	for y := 0; y+itemHeight <= gridHeight; y++ {
		for x := 0; x+itemWidth <= gridWidth; x++ {
			if regionFree(occupied, x, y, itemWidth, itemHeight) {
				return x, y, true
			}
		}
	}
	return 0, 0, false
}

func regionFree(occupied [][]bool, startX, startY, width, height int) bool {
	for y := startY; y < startY+height; y++ {
		if y < 0 || y >= len(occupied) {
			return false
		}
		for x := startX; x < startX+width; x++ {
			if x < 0 || x >= len(occupied[y]) {
				return false
			}
			if occupied[y][x] {
				return false
			}
		}
	}
	return true
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
	return normalizeItemRequest(uuid.New().String(), gameID, userID, req)
}

func normalizeItemRequest(itemID, gameID, createdByID string, req createItemRequest) (*models.Item, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("Item name cannot be empty")
	}
	if len(name) > maxItemNameLength {
		return nil, fmt.Errorf("Item name must be %d characters or less", maxItemNameLength)
	}

	description, err := normalizeItemDescription(req.Description)
	if err != nil {
		return nil, err
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
		CreatedByID:        createdByID,
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

func normalizeItemDescription(value string) (string, error) {
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	normalized = strings.TrimSpace(normalized)

	if strings.Count(normalized, "\n") > maxItemDescriptionLineBreaks {
		return "", fmt.Errorf("Item description can contain at most one line break")
	}
	if len(normalized) > maxItemDescriptionLength {
		return "", fmt.Errorf("Item description must be %d characters or less", maxItemDescriptionLength)
	}

	return normalized, nil
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
