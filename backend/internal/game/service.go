package game

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"backend/internal/models"

	"github.com/google/uuid"
)

type Service struct {
	repo      *Repository
	uploadDir string
}

func NewService(repo *Repository, uploadDir string) *Service {
	return &Service{repo: repo, uploadDir: uploadDir}
}

func (s *Service) GetUserGames(userID string) ([]models.Game, error) {
	return s.repo.GetUserGames(userID)
}

func generateInviteCode() string {
	b := make([]byte, 5)
	rand.Read(b)
	return strings.ToUpper(hex.EncodeToString(b))
}

func (s *Service) CreateGame(userID string, title, description, system string, maxPlayers int, showStandardAttrs, enableChat, enableItemTrading bool) (*models.Game, error) {
	plan, err := s.repo.GetUserPlan(userID)
	if err != nil {
		return nil, errors.New("failed to load plan")
	}

	count, err := s.repo.CountUserGames(userID)
	if err != nil {
		return nil, errors.New("failed to check game count")
	}
	if count >= int64(plan.MaxGamesOwned) {
		return nil, errors.New("game limit reached for your plan")
	}

	if len(title) < 1 || len(title) > 200 {
		return nil, errors.New("title must be 1-200 characters")
	}
	if maxPlayers < 1 {
		maxPlayers = 6
	}
	if maxPlayers > plan.MaxPlayersPerGame {
		maxPlayers = plan.MaxPlayersPerGame
	}

	expiresAt := time.Now().Add(15 * time.Minute)
	game := &models.Game{
		ID:                  uuid.New().String(),
		OwnerID:             userID,
		Title:               title,
		Description:         description,
		System:              system,
		InviteCode:          generateInviteCode(),
		InviteCodeExpiresAt: &expiresAt,
		MaxPlayers:          maxPlayers,
		ShowStandardAttrs:   showStandardAttrs,
		EnableChat:          enableChat,
		EnableItemTrading:   enableItemTrading,
	}

	if err := s.repo.CreateGame(game); err != nil {
		return nil, errors.New("failed to create game")
	}

	member := &models.GameMember{
		GameID:   game.ID,
		UserID:   userID,
		Role:     "gm",
		JoinedAt: time.Now(),
	}
	if err := s.repo.AddMember(member); err != nil {
		return nil, errors.New("failed to add owner as GM")
	}

	return game, nil
}

func (s *Service) JoinByCode(userID, code string) (*models.Game, error) {
	plan, err := s.repo.GetUserPlan(userID)
	if err != nil {
		return nil, errors.New("failed to load plan")
	}

	count, err := s.repo.CountUserGames(userID)
	if err != nil {
		return nil, errors.New("failed to check game count")
	}
	if count >= int64(plan.MaxGamesOwned) {
		return nil, errors.New("game limit reached for your plan")
	}

	game, err := s.repo.GetGameByInviteCode(code)
	if err != nil {
		return nil, errors.New("invalid invite code")
	}

	if game.InviteCodeExpiresAt != nil && time.Now().After(*game.InviteCodeExpiresAt) {
		return nil, errors.New("invite code has expired")
	}

	isMember, _ := s.repo.IsMember(game.ID, userID)
	if isMember {
		return nil, errors.New("you are already a member of this game")
	}

	memberCount, _ := s.repo.MemberCount(game.ID)
	if memberCount >= int64(game.MaxPlayers) {
		return nil, errors.New("this game is full")
	}

	member := &models.GameMember{
		GameID:   game.ID,
		UserID:   userID,
		Role:     "player",
		JoinedAt: time.Now(),
	}
	if err := s.repo.AddMember(member); err != nil {
		return nil, errors.New("failed to join game")
	}

	return game, nil
}

func (s *Service) RegenerateInviteCode(userID, gameID string) (string, time.Time, error) {
	game, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return "", time.Time{}, errors.New("game not found")
	}
	if game.OwnerID != userID {
		return "", time.Time{}, errors.New("only the game owner can regenerate the invite code")
	}

	newCode := generateInviteCode()
	expiresAt := time.Now().Add(15 * time.Minute)
	if err := s.repo.UpdateInviteCode(gameID, newCode, expiresAt); err != nil {
		return "", time.Time{}, errors.New("failed to update invite code")
	}
	return newCode, expiresAt, nil
}

func (s *Service) LeaveGame(userID, gameID string) error {
	game, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return errors.New("game not found")
	}
	if game.OwnerID == userID {
		return errors.New("game owner cannot leave, delete the game instead")
	}
	isMember, _ := s.repo.IsMember(gameID, userID)
	if !isMember {
		return errors.New("you are not a member of this game")
	}
	return s.repo.RemoveMember(gameID, userID)
}

func (s *Service) DeleteGame(userID, gameID string) error {
	game, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return errors.New("game not found")
	}
	if game.OwnerID != userID {
		return errors.New("only the game owner can delete the game")
	}
	if game.CoverImageID != nil {
		upload, err := s.repo.GetUploadByID(*game.CoverImageID)
		if err == nil {
			os.Remove(filepath.Join(s.uploadDir, upload.StorageKey))
			s.repo.SubtractStorageUsage(userID, upload.SizeBytes)
			s.repo.DeleteUpload(upload.ID)
		}
	}
	return s.repo.DeleteGame(gameID)
}

func (s *Service) UploadCoverImage(userID, gameID string, file multipart.File, header *multipart.FileHeader) (string, error) {
	game, err := s.repo.GetGameByID(gameID)
	if err != nil {
		return "", errors.New("game not found")
	}
	if game.OwnerID != userID {
		return "", errors.New("only the game owner can upload a cover image")
	}

	usage, _ := s.repo.GetStorageUsage(userID)
	plan, _ := s.repo.GetUserPlan(userID)
	var usedBytes int64
	if usage != nil {
		usedBytes = usage.UsedBytes
	}
	limitBytes := int64(plan.StorageLimitMB) * 1024 * 1024
	if usedBytes+header.Size > limitBytes {
		return "", errors.New("storage limit exceeded")
	}

	contentType := header.Header.Get("Content-Type")
	allowed := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
	}
	ext, ok := allowed[contentType]
	if !ok {
		return "", errors.New("only JPEG, PNG, and WebP images are allowed")
	}
	if header.Size > 5*1024*1024 {
		return "", errors.New("file must be under 5MB")
	}

	coverDir := filepath.Join(s.uploadDir, "covers")
	os.MkdirAll(coverDir, 0755)

	filename := fmt.Sprintf("cover_%s%s", uuid.New().String(), ext)
	dst, err := os.Create(filepath.Join(coverDir, filename))
	if err != nil {
		return "", errors.New("failed to save file")
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", errors.New("failed to save file")
	}

	if game.CoverImageID != nil {
		oldUpload, err := s.repo.GetUploadByID(*game.CoverImageID)
		if err == nil {
			os.Remove(filepath.Join(s.uploadDir, oldUpload.StorageKey))
			s.repo.SubtractStorageUsage(userID, oldUpload.SizeBytes)
			s.repo.DeleteUpload(oldUpload.ID)
		}
	}

	upload := &models.Upload{
		ID:           uuid.New().String(),
		UserID:       userID,
		StorageKey:   filepath.Join("covers", filename),
		OriginalName: header.Filename,
		FileType:     "game_cover",
		SizeBytes:    header.Size,
		MimeType:     contentType,
	}
	if err := s.repo.CreateUpload(upload); err != nil {
		return "", errors.New("failed to save upload record")
	}

	s.repo.UpdateCoverImage(gameID, &upload.ID)
	s.repo.AddStorageUsage(userID, header.Size)
	return upload.ID, nil
}
