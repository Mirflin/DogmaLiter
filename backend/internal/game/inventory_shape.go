package game

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"backend/internal/auth"
	"backend/internal/models"
	"backend/internal/realtime"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const maxInventoryPresetsPerGame = 50
const maxInventoryPresetNameLength = 60

// normalizeInventoryMask checks a backpack shape against its grid. An empty mask
// is the plain rectangle and passes as is; a shaped one needs one '0'/'1' per
// cell and at least one open cell.
func normalizeInventoryMask(mask string, width, height int) (string, error) {
	mask = strings.TrimSpace(mask)
	if mask == "" {
		return "", nil
	}
	if len(mask) != width*height {
		return "", fmt.Errorf("Backpack shape must have %d cells for a %dx%d grid", width*height, width, height)
	}
	open := 0
	for _, cell := range mask {
		switch cell {
		case '1':
			open++
		case '0':
		default:
			return "", fmt.Errorf("Backpack shape may only contain 0 and 1")
		}
	}
	if open == 0 {
		return "", fmt.Errorf("Backpack shape needs at least one open cell")
	}
	return mask, nil
}

// resizeInventoryMask keeps a shape anchored top-left when the grid is resized:
// cells that stay inside keep their state, new cells start closed.
func resizeInventoryMask(mask string, oldWidth, oldHeight, newWidth, newHeight int) string {
	if mask == "" || len(mask) != oldWidth*oldHeight {
		return ""
	}
	var resized strings.Builder
	resized.Grow(newWidth * newHeight)
	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			if x < oldWidth && y < oldHeight {
				resized.WriteByte(mask[y*oldWidth+x])
			} else {
				resized.WriteByte('0')
			}
		}
	}
	return resized.String()
}

func inventoryCellOpen(character *models.Character, x, y int) bool {
	if x < 0 || y < 0 || x >= character.InventoryWidth || y >= character.InventoryHeight {
		return false
	}
	if character.InventoryMask == "" {
		return true
	}
	index := y*character.InventoryWidth + x
	return index < len(character.InventoryMask) && character.InventoryMask[index] == '1'
}

// inventoryRegionOpen reports whether a width x height footprint at (x, y) lies
// inside the grid and only on open cells. It ignores other items.
func inventoryRegionOpen(character *models.Character, x, y, width, height int) bool {
	for cy := y; cy < y+height; cy++ {
		for cx := x; cx < x+width; cx++ {
			if !inventoryCellOpen(character, cx, cy) {
				return false
			}
		}
	}
	return true
}

// newInventoryOccupancy builds the occupancy grid for auto-placement: closed
// cells and cells under existing items count as taken.
func newInventoryOccupancy(character *models.Character) [][]bool {
	occupied := make([][]bool, character.InventoryHeight)
	for y := range occupied {
		occupied[y] = make([]bool, character.InventoryWidth)
		for x := range occupied[y] {
			occupied[y][x] = !inventoryCellOpen(character, x, y)
		}
	}
	markInventoryOccupancy(occupied, character.Inventory)
	return occupied
}

func serializeInventoryPreset(preset *models.InventoryPreset) map[string]interface{} {
	return map[string]interface{}{
		"id":         preset.ID,
		"game_id":    preset.GameID,
		"name":       preset.Name,
		"width":      preset.Width,
		"height":     preset.Height,
		"mask":       preset.Mask,
		"created_at": preset.CreatedAt,
		"updated_at": preset.UpdatedAt,
	}
}

type inventoryPresetRequest struct {
	Name   string `json:"name"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Mask   string `json:"mask"`
}

func normalizeInventoryPresetRequest(req inventoryPresetRequest) (string, int, int, string, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return "", 0, 0, "", fmt.Errorf("Preset name cannot be empty")
	}
	if utf8.RuneCountInString(name) > maxInventoryPresetNameLength {
		return "", 0, 0, "", fmt.Errorf("Preset name must be %d characters or less", maxInventoryPresetNameLength)
	}
	width, err := validateInventoryDimension(req.Width, "Width")
	if err != nil {
		return "", 0, 0, "", err
	}
	height, err := validateInventoryDimension(req.Height, "Height")
	if err != nil {
		return "", 0, 0, "", err
	}
	if strings.TrimSpace(req.Mask) == "" {
		return "", 0, 0, "", fmt.Errorf("Draw at least one open cell")
	}
	mask, err := normalizeInventoryMask(req.Mask, width, height)
	if err != nil {
		return "", 0, 0, "", err
	}
	return name, width, height, mask, nil
}

// authorizeGM resolves the game and refuses anyone but its GM. It writes the
// error response itself and returns false when the caller should stop.
func (h *Handler) authorizeGM(w http.ResponseWriter, r *http.Request, gameID string) bool {
	_, isGM, err := h.authorizeGameAccess(r, gameID)
	if err != nil {
		h.respondGameAccessError(w, gameID, err)
		return false
	}
	if !isGM {
		respondJSON(w, 403, map[string]string{"error": "Only the GM can manage backpack presets"})
		return false
	}
	return true
}

func (h *Handler) ListInventoryPresets(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	if !h.authorizeGM(w, r, gameID) {
		return
	}

	presets, err := h.service.repo.ListInventoryPresets(gameID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to load backpack presets"})
		return
	}

	result := make([]map[string]interface{}, 0, len(presets))
	for i := range presets {
		result = append(result, serializeInventoryPreset(&presets[i]))
	}
	respondJSON(w, 200, map[string]interface{}{"presets": result})
}

func (h *Handler) CreateInventoryPreset(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	if !h.authorizeGM(w, r, gameID) {
		return
	}

	var req inventoryPresetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}
	name, width, height, mask, err := normalizeInventoryPresetRequest(req)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	count, err := h.service.repo.CountInventoryPresets(gameID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to check preset limit"})
		return
	}
	if count >= maxInventoryPresetsPerGame {
		respondJSON(w, 400, map[string]string{"error": fmt.Sprintf("A game can keep up to %d backpack presets", maxInventoryPresetsPerGame)})
		return
	}

	preset := &models.InventoryPreset{
		ID:     uuid.New().String(),
		GameID: gameID,
		Name:   name,
		Width:  width,
		Height: height,
		Mask:   mask,
	}
	if err := h.service.repo.CreateInventoryPreset(preset); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to save backpack preset"})
		return
	}
	respondJSON(w, 201, map[string]interface{}{"preset": serializeInventoryPreset(preset)})
}

func (h *Handler) UpdateInventoryPreset(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	presetID := chi.URLParam(r, "presetID")
	if !h.authorizeGM(w, r, gameID) {
		return
	}

	preset, err := h.service.repo.GetInventoryPreset(gameID, presetID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "Backpack preset not found"})
		return
	}

	var req inventoryPresetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}
	name, width, height, mask, err := normalizeInventoryPresetRequest(req)
	if err != nil {
		respondJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}

	preset.Name = name
	preset.Width = width
	preset.Height = height
	preset.Mask = mask
	if err := h.service.repo.UpdateInventoryPreset(preset); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to save backpack preset"})
		return
	}
	respondJSON(w, 200, map[string]interface{}{"preset": serializeInventoryPreset(preset)})
}

func (h *Handler) DeleteInventoryPreset(w http.ResponseWriter, r *http.Request) {
	gameID := chi.URLParam(r, "gameID")
	presetID := chi.URLParam(r, "presetID")
	if !h.authorizeGM(w, r, gameID) {
		return
	}

	affected, err := h.service.repo.DeleteInventoryPreset(gameID, presetID)
	if err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to delete backpack preset"})
		return
	}
	if affected == 0 {
		respondJSON(w, 404, map[string]string{"error": "Backpack preset not found"})
		return
	}
	respondJSON(w, 200, map[string]interface{}{"deleted": true})
}

// ApplyInventoryPreset copies a preset's shape onto several characters at once.
// Either every character gets it or none does: if anyone's items would end up
// outside the new shape, nothing changes and the response names who blocked it.
func (h *Handler) ApplyInventoryPreset(w http.ResponseWriter, r *http.Request) {
	userID := auth.GetUserID(r)
	gameID := chi.URLParam(r, "gameID")
	presetID := chi.URLParam(r, "presetID")
	if !h.authorizeGM(w, r, gameID) {
		return
	}

	preset, err := h.service.repo.GetInventoryPreset(gameID, presetID)
	if err != nil {
		respondJSON(w, 404, map[string]string{"error": "Backpack preset not found"})
		return
	}

	var req struct {
		CharacterIDs []string `json:"character_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, 400, map[string]string{"error": "Invalid request body"})
		return
	}
	if len(req.CharacterIDs) == 0 {
		respondJSON(w, 400, map[string]string{"error": "Choose at least one character"})
		return
	}
	if len(req.CharacterIDs) > 100 {
		respondJSON(w, 400, map[string]string{"error": "You can apply a preset to up to 100 characters at once"})
		return
	}

	characters := make([]*models.Character, 0, len(req.CharacterIDs))
	blocked := make([]string, 0)
	seen := make(map[string]bool, len(req.CharacterIDs))
	for _, characterID := range req.CharacterIDs {
		if seen[characterID] {
			continue
		}
		seen[characterID] = true

		character, err := h.service.repo.GetCharacterByID(gameID, characterID)
		if err != nil {
			respondJSON(w, 404, map[string]string{"error": fmt.Sprintf("Character %s not found", characterID)})
			return
		}
		character.InventoryWidth = preset.Width
		character.InventoryHeight = preset.Height
		character.InventoryMask = preset.Mask
		if err := validateInventoryResize(character); err != nil {
			blocked = append(blocked, fmt.Sprintf("%s (%s)", character.Name, err.Error()))
			continue
		}
		characters = append(characters, character)
	}

	if len(blocked) > 0 {
		respondJSON(w, 400, map[string]string{
			"error": fmt.Sprintf("Move items out of the way first: %s", strings.Join(blocked, "; ")),
		})
		return
	}

	if err := h.service.repo.ApplyInventoryShape(gameID, characters); err != nil {
		respondJSON(w, 500, map[string]string{"error": "Failed to apply backpack preset"})
		return
	}

	for _, character := range characters {
		h.logActivity(gameID, userID, character.Name, "Backpack changed", preset.Name)
		h.broadcast(gameID, realtime.EventCharacterUpdated, map[string]interface{}{"character_id": character.ID})
	}
	respondJSON(w, 200, map[string]interface{}{"applied": len(characters)})
}
