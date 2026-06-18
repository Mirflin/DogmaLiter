package game

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/realtime"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func serializeTradeOffer(offer *models.TradeOffer) map[string]interface{} {
	items := make([]map[string]interface{}, 0, len(offer.Items))
	for _, it := range offer.Items {
		items = append(items, map[string]interface{}{
			"id":             it.ID,
			"item_id":        it.ItemID,
			"quantity":       it.Quantity,
			"durability":     it.Durability,
			"max_durability": it.MaxDurability,
			"enchantment":    it.Enchantment,
			"item":           serializeItem(it.Item),
		})
	}
	return map[string]interface{}{
		"id":                  offer.ID,
		"status":              offer.Status,
		"created_at":          offer.CreatedAt,
		"from_user_id":        offer.FromUserID,
		"from_username":       offer.FromUsername,
		"from_character_id":   offer.FromCharacterID,
		"from_character_name": offer.FromCharacterName,
		"to_user_id":          offer.ToUserID,
		"to_username":         offer.ToUsername,
		"to_character_id":     offer.ToCharacterID,
		"to_character_name":   offer.ToCharacterName,
		"items":               items,
	}
}

func placeTradeItems(character *models.Character, items []models.TradeOfferItem) ([]models.CharacterInventory, bool) {
	width := character.InventoryWidth
	height := character.InventoryHeight
	occupied := make([][]bool, height)
	for y := range occupied {
		occupied[y] = make([]bool, width)
	}
	markInventoryOccupancy(occupied, character.Inventory)

	entries := make([]models.CharacterInventory, 0, len(items))
	allPlaced := true
	for _, it := range items {
		itemWidth := it.Item.GridWidth
		itemHeight := it.Item.GridHeight
		if itemWidth < 1 {
			itemWidth = 1
		}
		if itemHeight < 1 {
			itemHeight = 1
		}
		gridX, gridY, placed := findFreeInventorySlot(occupied, width, height, itemWidth, itemHeight)
		if placed {
			markCells(occupied, gridX, gridY, itemWidth, itemHeight, true)
		} else {
			allPlaced = false
			gridX, gridY = 0, 0
		}
		entries = append(entries, models.CharacterInventory{
			ID:            uuid.New().String(),
			CharacterID:   character.ID,
			ItemID:        it.ItemID,
			Quantity:      it.Quantity,
			Durability:    copyIntPointer(it.Durability),
			MaxDurability: copyIntPointer(it.MaxDurability),
			Enchantment:   it.Enchantment,
			GridX:         gridX,
			GridY:         gridY,
			IsRotated:     false,
		})
	}
	return entries, allPlaced
}

func (h *Handler) CreateTrade(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	game, _, err := h.authorizeGameAccess(r, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}
	if !game.EnableItemTrading {
		respondJSON(w, 403, map[string]string{"error": "Item trading is disabled for this game"})
		return
	}

	var req struct {
		FromCharacterID  string   `json:"from_character_id"`
		ToCharacterID    string   `json:"to_character_id"`
		InventoryItemIDs []string `json:"inventory_item_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}
	if req.FromCharacterID == "" || req.ToCharacterID == "" {
		respondJSON(w, 400, map[string]string{"error": "Both characters are required"})
		return
	}
	if req.FromCharacterID == req.ToCharacterID {
		respondJSON(w, 400, map[string]string{"error": "Choose a different character to trade with"})
		return
	}
	if len(req.InventoryItemIDs) == 0 {
		respondJSON(w, 400, map[string]string{"error": "Select at least one item to offer"})
		return
	}
	if len(req.InventoryItemIDs) > 30 {
		respondJSON(w, 400, map[string]string{"error": "You can offer up to 30 items at once"})
		return
	}

	from, err := h.service.repo.GetCharacterByID(gameID, req.FromCharacterID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "Your character was not found"})
		return
	}
	if from.UserID != userID {
		respondJSON(w, 403, map[string]string{"error": "You can only offer items from your own character"})
		return
	}

	to, err := h.service.repo.GetCharacterByID(gameID, req.ToCharacterID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "The recipient character was not found"})
		return
	}

	equipped := make(map[string]bool)
	for _, eq := range from.Equipment {
		equipped[eq.InventoryItemID] = true
	}
	inventoryByID := make(map[string]models.CharacterInventory, len(from.Inventory))
	for _, inv := range from.Inventory {
		inventoryByID[inv.ID] = inv
	}

	offerID := uuid.New().String()
	items := make([]models.TradeOfferItem, 0, len(req.InventoryItemIDs))
	removeIDs := make([]string, 0, len(req.InventoryItemIDs))
	seen := make(map[string]bool)
	for _, inventoryItemID := range req.InventoryItemIDs {
		if seen[inventoryItemID] {
			continue
		}
		seen[inventoryItemID] = true
		inv, ok := inventoryByID[inventoryItemID]
		if !ok {
			respondJSON(w, 400, map[string]string{"error": "One of the selected items is not in your inventory"})
			return
		}
		if equipped[inventoryItemID] {
			respondJSON(w, 400, map[string]string{"error": "Unequip items before offering them in a trade"})
			return
		}
		items = append(items, models.TradeOfferItem{
			ID:            uuid.New().String(),
			TradeOfferID:  offerID,
			ItemID:        inv.ItemID,
			Quantity:      inv.Quantity,
			Durability:    copyIntPointer(inv.Durability),
			MaxDurability: copyIntPointer(inv.MaxDurability),
			Enchantment:   inv.Enchantment,
		})
		removeIDs = append(removeIDs, inventoryItemID)
	}

	offer := &models.TradeOffer{
		ID:                offerID,
		GameID:            gameID,
		FromUserID:        userID,
		FromCharacterID:   from.ID,
		FromUsername:      from.User.Username,
		FromCharacterName: from.Name,
		ToUserID:          to.UserID,
		ToCharacterID:     to.ID,
		ToUsername:        to.User.Username,
		ToCharacterName:   to.Name,
		Status:            "pending",
		CreatedAt:         time.Now(),
		Items:             items,
	}

	if err := h.service.repo.CreateTradeOffer(offer, removeIDs); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to create the trade offer"})
		return
	}

	h.logActivity(gameID, userID, from.Name, "Offered trade", fmt.Sprintf("%d item(s) to %s", len(items), to.Name))
	h.broadcast(gameID, realtime.EventTradesChanged, nil)
	h.broadcast(gameID, realtime.EventCharacterUpdated, map[string]interface{}{"character_id": from.ID})
	respondJSON(w, 201, map[string]interface{}{"offer": serializeTradeOffer(offer)})
}

func (h *Handler) ListTrades(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")

	if _, _, err := h.authorizeGameAccess(r, gameID); err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	offers, err := h.service.repo.ListTradeOffers(gameID, userID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load trades"})
		return
	}

	incoming := make([]map[string]interface{}, 0)
	outgoing := make([]map[string]interface{}, 0)
	for i := range offers {
		serialized := serializeTradeOffer(&offers[i])
		if offers[i].ToUserID == userID {
			incoming = append(incoming, serialized)
		} else {
			outgoing = append(outgoing, serialized)
		}
	}

	characters, err := h.service.repo.ListGameCharacters(gameID, nil)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load characters"})
		return
	}
	targets := make([]map[string]interface{}, 0, len(characters))
	for _, character := range characters {
		targets = append(targets, map[string]interface{}{
			"id":      character.ID,
			"name":    character.Name,
			"user_id": character.UserID,
			"owner":   character.User.Username,
		})
	}

	respondJSON(w, 200, map[string]interface{}{
		"incoming": incoming,
		"outgoing": outgoing,
		"targets":  targets,
	})
}

func (h *Handler) AcceptTrade(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	tradeID := chi.URLParam(r, "tradeID")

	if _, _, err := h.authorizeGameAccess(r, gameID); err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	offer, err := h.service.repo.GetTradeOffer(gameID, tradeID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "Trade offer not found"})
		return
	}
	if offer.Status != "pending" {
		respondJSON(w, 400, map[string]string{"error": "This offer is no longer pending"})
		return
	}
	if offer.ToUserID != userID {
		respondJSON(w, 403, map[string]string{"error": "Only the recipient can accept this offer"})
		return
	}

	to, err := h.service.repo.GetCharacterByID(gameID, offer.ToCharacterID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "The recipient character no longer exists"})
		return
	}

	entries, allPlaced := placeTradeItems(to, offer.Items)
	if !allPlaced {
		respondJSON(w, 400, map[string]string{"error": "Not enough inventory space to accept this trade"})
		return
	}

	if err := h.service.repo.ResolveTradeOffer(offer.ID, "accepted", entries); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to accept the trade"})
		return
	}

	h.logActivity(gameID, userID, to.Name, "Accepted trade", fmt.Sprintf("%d item(s) from %s", len(entries), offer.FromCharacterName))
	h.broadcast(gameID, realtime.EventTradesChanged, nil)
	h.broadcast(gameID, realtime.EventCharacterUpdated, map[string]interface{}{"character_id": to.ID})
	respondJSON(w, 200, map[string]interface{}{"success": true})
}

func (h *Handler) DeclineTrade(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	tradeID := chi.URLParam(r, "tradeID")

	if _, _, err := h.authorizeGameAccess(r, gameID); err != nil {
		h.respondGameAccessError(w, gameID, err)
		return
	}

	offer, err := h.service.repo.GetTradeOffer(gameID, tradeID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "Trade offer not found"})
		return
	}
	if offer.Status != "pending" {
		respondJSON(w, 400, map[string]string{"error": "This offer is no longer pending"})
		return
	}
	if offer.ToUserID != userID && offer.FromUserID != userID {
		respondJSON(w, 403, map[string]string{"error": "You are not part of this trade"})
		return
	}

	status := "declined"
	if offer.FromUserID == userID {
		status = "cancelled"
	}

	from, err := h.service.repo.GetCharacterByID(gameID, offer.FromCharacterID)
	if err != nil {
		if err := h.service.repo.ResolveTradeOffer(offer.ID, status, nil); err != nil {
			respondJSON(w, 500, map[string]string{"error": "Failed to resolve the trade"})
			return
		}
		h.broadcast(gameID, realtime.EventTradesChanged, nil)
		respondJSON(w, 200, map[string]interface{}{"success": true})
		return
	}

	entries, _ := placeTradeItems(from, offer.Items)
	if err := h.service.repo.ResolveTradeOffer(offer.ID, status, entries); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to resolve the trade"})
		return
	}

	action := "Declined trade"
	if status == "cancelled" {
		action = "Cancelled trade"
	}
	h.logActivity(gameID, userID, offer.FromCharacterName, action, fmt.Sprintf("%d item(s)", len(offer.Items)))
	h.broadcast(gameID, realtime.EventTradesChanged, nil)
	h.broadcast(gameID, realtime.EventCharacterUpdated, map[string]interface{}{"character_id": from.ID})
	respondJSON(w, 200, map[string]interface{}{"success": true})
}
