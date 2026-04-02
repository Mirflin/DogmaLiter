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
