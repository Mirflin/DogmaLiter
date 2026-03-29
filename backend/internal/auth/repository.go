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
