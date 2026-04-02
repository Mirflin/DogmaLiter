package news

import (
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

func (s *Service) Create(authorID, title, content string, imageFile multipart.File, imageHeader *multipart.FileHeader) (*models.NewsPost, error) {
	if len(title) < 1 || len(title) > 300 {
		return nil, errors.New("title must be 1-300 characters")
	}
	if len(content) < 1 {
		return nil, errors.New("content is required")
	}

	var imageID *string
	if imageFile != nil && imageHeader != nil {
		ct := imageHeader.Header.Get("Content-Type")
		if !strings.HasPrefix(ct, "image/") {
			return nil, errors.New("only image files are allowed")
		}
		if imageHeader.Size > 5*1024*1024 {
			return nil, errors.New("image must be under 5MB")
		}

		ext := filepath.Ext(imageHeader.Filename)
		storageKey := fmt.Sprintf("news/%s%s", uuid.New().String(), ext)

		dir := filepath.Join(s.uploadDir, "news")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, errors.New("failed to create upload directory")
		}

		dst, err := os.Create(filepath.Join(s.uploadDir, storageKey))
		if err != nil {
			return nil, errors.New("failed to save image")
		}
		defer dst.Close()

		if _, err := io.Copy(dst, imageFile); err != nil {
			return nil, errors.New("failed to write image")
		}

		upload := &models.Upload{
			ID:           uuid.New().String(),
			UserID:       authorID,
			FileType:     "news_image",
			OriginalName: imageHeader.Filename,
			StorageKey:   storageKey,
			MimeType:     ct,
			SizeBytes:    imageHeader.Size,
			CreatedAt:    time.Now(),
		}
		if err := s.repo.CreateUpload(upload); err != nil {
			return nil, errors.New("failed to save upload record")
		}
		s.repo.AddStorageUsage(authorID, imageHeader.Size)
		imageID = &upload.ID
	}

	now := time.Now()
	post := &models.NewsPost{
		ID:          uuid.New().String(),
		AuthorID:    authorID,
		Title:       title,
		Content:     content,
		ImageID:     imageID,
		IsPublished: true,
		PublishedAt: &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(post); err != nil {
		return nil, errors.New("failed to create news post")
	}

	created, err := s.repo.GetByID(post.ID)
	if err != nil {
		return post, nil
	}
	return created, nil
}

func (s *Service) GetByID(id string) (*models.NewsPost, error) {
	post, err := s.repo.GetByID(id)
	if err != nil {
		return nil, errors.New("news post not found")
	}
	if !post.IsPublished {
		return nil, errors.New("news post not found")
	}
	return post, nil
}

func (s *Service) ListPublished(limit, offset int) ([]models.NewsPost, int64, error) {
	if limit <= 0 || limit > 50 {
		limit = 12
	}
	if offset < 0 {
		offset = 0
	}

	posts, err := s.repo.ListPublished(limit, offset)
	if err != nil {
		return nil, 0, errors.New("failed to load news")
	}

	total, err := s.repo.CountPublished()
	if err != nil {
		return nil, 0, errors.New("failed to count news")
	}

	return posts, total, nil
}
