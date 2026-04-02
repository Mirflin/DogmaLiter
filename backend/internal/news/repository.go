package news

import (
	"backend/internal/models"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(post *models.NewsPost) error {
	return r.db.Create(post).Error
}

func (r *Repository) GetByID(id string) (*models.NewsPost, error) {
	var post models.NewsPost
	err := r.db.Preload("Author").Preload("Image").First(&post, "id = ?", id).Error
	return &post, err
}

func (r *Repository) ListPublished(limit, offset int) ([]models.NewsPost, error) {
	var posts []models.NewsPost
	err := r.db.Where("is_published = ?", true).
		Preload("Author").
		Preload("Image").
		Order("published_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&posts).Error
	return posts, err
}

func (r *Repository) CountPublished() (int64, error) {
	var count int64
	err := r.db.Model(&models.NewsPost{}).Where("is_published = ?", true).Count(&count).Error
	return count, err
}

func (r *Repository) CreateUpload(upload *models.Upload) error {
	return r.db.Create(upload).Error
}

func (r *Repository) AddStorageUsage(userID string, bytes int64) error {
	return r.db.Model(&models.UserStorageUsage{}).Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"used_bytes":  gorm.Expr("used_bytes + ?", bytes),
			"files_count": gorm.Expr("files_count + 1"),
		}).Error
}
