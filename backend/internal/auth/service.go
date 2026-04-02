package auth

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"backend/internal/models"
)

type Service struct {
	repo        *Repository
	jwt         *JWTManager
	mailer      *Mailer
	frontendURL string
	uploadDir   string
}

func NewService(repo *Repository, jwt *JWTManager, mailer *Mailer, frontendURL string, uploadDir string) *Service {
	return &Service{
		repo:        repo,
		jwt:         jwt,
		mailer:      mailer,
		frontendURL: frontendURL,
		uploadDir:   uploadDir,
	}
}

type RegisterInput struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

func (s *Service) Register(input RegisterInput) (*RegisterResponse, error) {
	if len(input.Username) < 3 || len(input.Username) > 50 {
		return nil, fmt.Errorf("username needs to be between 3 and 50 characters")
	}
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}
	if input.Email == "" {
		return nil, fmt.Errorf("email is required")
	}

	if existing, _ := s.repo.GetUserByEmail(input.Email); existing != nil {
		return nil, fmt.Errorf("email is already registered")
	}
	if existing, _ := s.repo.GetUserByUsername(input.Username); existing != nil {
		return nil, fmt.Errorf("username is already taken")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}
	hashStr := string(hash)

	user := &models.User{
		ID:           uuid.New().String(),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: &hashStr,
		PlanID:       "free",
		IsVerified:   false,
	}

	if err := s.repo.CreateUser(user); err != nil {
		return nil, fmt.Errorf("error creating user: %w", err)
	}

	s.repo.CreateStorageUsage(user.ID)

	verifyToken := &models.VerificationToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Token:     uuid.New().String(),
		Type:      "email_verify",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.repo.CreateVerificationToken(verifyToken); err != nil {
		return nil, fmt.Errorf("error creating verification token: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify?token=%s", s.frontendURL, verifyToken.Token)
	go func() {
		if err := s.mailer.SendVerificationEmail(user.Email, user.Username, verifyURL); err != nil {
			fmt.Printf("Error sending email to %s: %v\n", user.Email, err)
		}
	}()

	return &RegisterResponse{
		Message: "Registration successful! Please check your email to verify your account.",
		UserID:  user.ID,
	}, nil
}

func (s *Service) VerifyEmail(token string) error {
	vt, err := s.repo.GetVerificationToken(token, "email_verify")
	if err != nil {
		return fmt.Errorf("invalid or expired link")
	}

	if err := s.repo.VerifyUser(vt.UserID); err != nil {
		return fmt.Errorf("verification error: %w", err)
	}

	s.repo.DeleteVerificationToken(vt.ID)

	return nil
}

func (s *Service) ResendVerification(email string) error {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return fmt.Errorf("user not found")
	}
	if user.IsVerified {
		return fmt.Errorf("email is already verified")
	}

	s.repo.DeleteUserVerificationTokens(user.ID, "email_verify")

	verifyToken := &models.VerificationToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Token:     uuid.New().String(),
		Type:      "email_verify",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	if err := s.repo.CreateVerificationToken(verifyToken); err != nil {
		return fmt.Errorf("error creating verification token: %w", err)
	}

	verifyURL := fmt.Sprintf("%s/verify?token=%s", s.frontendURL, verifyToken.Token)
	go func() {
		if err := s.mailer.SendVerificationEmail(user.Email, user.Username, verifyURL); err != nil {
			fmt.Printf("Error sending email to %s: %v\n", user.Email, err)
		}
	}()

	return nil
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	User         UserResponse `json:"user"`
}

type UserResponse struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	Email      string  `json:"email"`
	Role       string  `json:"role"`
	AvatarID   *string `json:"avatar_id"`
	PlanID     string  `json:"plan_id"`
	IsVerified bool    `json:"is_verified"`
}

func (s *Service) Login(input LoginInput) (*LoginResponse, error) {
	user, err := s.repo.GetUserByEmail(input.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("invalid email or password")
		}
		return nil, fmt.Errorf("error fetching user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.Password)); err != nil {
		return nil, fmt.Errorf("invalid email or password")
	}

	if !user.IsVerified {
		return nil, fmt.Errorf("email not verified, please check your email")
	}

	tokens, err := s.jwt.GenerateTokenPair(user.ID, user.Username, user.Role)
	if err != nil {
		return nil, fmt.Errorf("error generating tokens: %w", err)
	}

	return &LoginResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		User: UserResponse{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Role:       user.Role,
			AvatarID:   user.AvatarID,
			PlanID:     user.PlanID,
			IsVerified: user.IsVerified,
		},
	}, nil
}

func (s *Service) RefreshToken(refreshToken string) (*TokenPair, error) {
	claims, err := s.jwt.ValidateToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token")
	}

	user, err := s.repo.GetUserByID(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return s.jwt.GenerateTokenPair(user.ID, user.Username, user.Role)
}

func (s *Service) RequestPasswordReset(email string) error {
	user, err := s.repo.GetUserByEmail(email)
	if err != nil {
		return nil
	}

	s.repo.DeleteUserVerificationTokens(user.ID, "password_reset")

	resetToken := &models.VerificationToken{
		ID:        uuid.New().String(),
		UserID:    user.ID,
		Token:     uuid.New().String(),
		Type:      "password_reset",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	if err := s.repo.CreateVerificationToken(resetToken); err != nil {
		return fmt.Errorf("error creating token: %w", err)
	}

	resetURL := fmt.Sprintf("%s/reset-password?token=%s", s.frontendURL, resetToken.Token)
	go func() {
		if err := s.mailer.SendPasswordResetEmail(user.Email, user.Username, resetURL); err != nil {
			fmt.Printf("Error sending password reset email to %s: %v\n", user.Email, err)
		}
	}()

	return nil
}

type ResetPasswordInput struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

func (s *Service) ResetPassword(input ResetPasswordInput) error {
	if len(input.NewPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	vt, err := s.repo.GetVerificationToken(input.Token, "password_reset")
	if err != nil {
		return fmt.Errorf("invalid or expired link")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}

	if err := s.repo.UpdatePassword(vt.UserID, string(hash)); err != nil {
		return fmt.Errorf("error updating password: %w", err)
	}

	s.repo.DeleteVerificationToken(vt.ID)

	return nil
}

type UpdateUsernameInput struct {
	Username string `json:"username"`
}

func (s *Service) UpdateUsername(userID string, input UpdateUsernameInput) error {
	if len(input.Username) < 3 || len(input.Username) > 50 {
		return fmt.Errorf("username needs to be between 3 and 50 characters")
	}

	existing, _ := s.repo.GetUserByUsername(input.Username)
	if existing != nil && existing.ID != userID {
		return fmt.Errorf("username is already taken")
	}

	return s.repo.UpdateUsername(userID, input.Username)
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (s *Service) ChangePassword(userID string, input ChangePasswordInput) error {
	if len(input.NewPassword) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}

	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.PasswordHash == nil {
		return fmt.Errorf("password change not available for this account")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(input.CurrentPassword)); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}

	return s.repo.UpdatePassword(userID, string(hash))
}

var allowedAvatarMimes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

const maxAvatarSize = 2 * 1024 * 1024 // 2MB

func (s *Service) UploadAvatar(userID string, file multipart.File, header *multipart.FileHeader) (*models.Upload, error) {
	if header.Size > maxAvatarSize {
		return nil, fmt.Errorf("file too large, maximum 2MB")
	}

	ext, ok := allowedAvatarMimes[header.Header.Get("Content-Type")]
	if !ok {
		return nil, fmt.Errorf("only JPEG, PNG, and WebP images are allowed")
	}

	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	userWithPlan, err := s.repo.GetUserWithPlan(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load plan")
	}
	if userWithPlan.StorageFrozen {
		return nil, fmt.Errorf("storage is frozen, please upgrade your plan to upload files")
	}
	usage, _ := s.repo.GetStorageUsage(userID)
	var usedBytes int64
	if usage != nil {
		usedBytes = usage.UsedBytes
	}
	limitBytes := int64(userWithPlan.Plan.StorageLimitMB) * 1024 * 1024
	if usedBytes+header.Size > limitBytes {
		return nil, fmt.Errorf("storage limit exceeded")
	}


	if user.AvatarID != nil {
		oldUpload, err := s.repo.GetUploadByID(*user.AvatarID)
		if err == nil {
			os.Remove(filepath.Join(s.uploadDir, oldUpload.StorageKey))
			s.repo.SubtractStorageUsage(userID, oldUpload.SizeBytes)
			s.repo.UpdateAvatarID(userID, nil)
			s.repo.DeleteUpload(oldUpload.ID)
		}
	}

	uploadID := uuid.New().String()
	storageKey := filepath.Join("avatars", uploadID+ext)
	fullPath := filepath.Join(s.uploadDir, storageKey)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("error creating directory: %w", err)
	}

	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("error saving file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("error writing file: %w", err)
	}

	upload := &models.Upload{
		ID:           uploadID,
		UserID:       userID,
		FileType:     "avatar",
		OriginalName: header.Filename,
		StorageKey:   storageKey,
		MimeType:     header.Header.Get("Content-Type"),
		SizeBytes:    header.Size,
	}

	if err := s.repo.CreateUpload(upload); err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("error saving upload record: %w", err)
	}

	if err := s.repo.UpdateAvatarID(userID, &uploadID); err != nil {
		return nil, fmt.Errorf("error linking avatar: %w", err)
	}

	s.repo.AddStorageUsage(userID, header.Size)

	return upload, nil
}

func (s *Service) DeleteAvatar(userID string) error {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return fmt.Errorf("user not found")
	}

	if user.AvatarID == nil {
		return fmt.Errorf("no avatar to remove")
	}

	upload, err := s.repo.GetUploadByID(*user.AvatarID)
	if err != nil {
		return fmt.Errorf("avatar not found")
	}

	if err := s.repo.UpdateAvatarID(userID, nil); err != nil {
		return fmt.Errorf("error unlinking avatar: %w", err)
	}

	os.Remove(filepath.Join(s.uploadDir, upload.StorageKey))
	s.repo.SubtractStorageUsage(userID, upload.SizeBytes)
	s.repo.DeleteUpload(upload.ID)

	return nil
}

func (s *Service) GetUploadByID(id string) (*models.Upload, error) {
	return s.repo.GetUploadByID(id)
}

func (s *Service) GetUploadDir() string {
	return s.uploadDir
}

type StorageUsageResponse struct {
	UsedBytes      int64 `json:"used_bytes"`
	FilesCount     int   `json:"files_count"`
	StorageLimitMB int   `json:"storage_limit_mb"`
}

func (s *Service) GetStorageUsage(userID string) (*StorageUsageResponse, error) {
	user, err := s.repo.GetUserWithPlan(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	usage, err := s.repo.GetStorageUsage(userID)
	if err != nil {
		return &StorageUsageResponse{
			UsedBytes:      0,
			FilesCount:     0,
			StorageLimitMB: user.Plan.StorageLimitMB,
		}, nil
	}

	return &StorageUsageResponse{
		UsedBytes:      usage.UsedBytes,
		FilesCount:     usage.FilesCount,
		StorageLimitMB: user.Plan.StorageLimitMB,
	}, nil
}

type ProfileResponse struct {
	ID         string  `json:"id"`
	Username   string  `json:"username"`
	Email      string  `json:"email"`
	Role       string  `json:"role"`
	AvatarID   *string `json:"avatar_id"`
	PlanName   string  `json:"plan_name"`
	IsVerified bool    `json:"is_verified"`
	CreatedAt  string  `json:"created_at"`
}

func (s *Service) GetProfile(userID string) (*ProfileResponse, error) {
	user, err := s.repo.GetUserWithPlan(userID)
	if err != nil {
		return nil, fmt.Errorf("user not found")
	}

	return &ProfileResponse{
		ID:         user.ID,
		Username:   user.Username,
		Email:      user.Email,
		Role:       user.Role,
		AvatarID:   user.AvatarID,
		PlanName:   user.Plan.Name,
		IsVerified: user.IsVerified,
		CreatedAt:  user.CreatedAt.Format("02.01.2006"),
	}, nil
}

func (s *Service) GetAllPlans() ([]models.Plan, error) {
	return s.repo.GetAllPlans()
}
