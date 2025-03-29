package repository

import (
	"errors"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"

	"gorm.io/gorm"
)

// ImageRepository 画像に関するデータベース操作を行うインターフェース
type ImageRepository interface {
	Create(image *models.Image) (uint, error)
	FindByID(id uint) (*models.Image, error)
	Update(image *models.Image) error
	Delete(id uint) error
	ListPendingImages(limit int) ([]models.Image, error)
	UpdateStatus(id uint, status string, webpPath string, errorMessage string) error
	CountPendingImages() (int64, error)
	UpdateImageStats(id uint, webpPath string, originalSize int64, webpSize int64, compressionRatio float64, width int, height int) error
}

// imageRepository ImageRepositoryの実装
type imageRepository struct {
	db *gorm.DB
}

// NewImageRepository ImageRepositoryを作成
func NewImageRepository(db *gorm.DB) ImageRepository {
	return &imageRepository{db: db}
}

// Create 新しい画像レコードを作成
func (r *imageRepository) Create(image *models.Image) (uint, error) {
	if err := r.db.Create(image).Error; err != nil {
		return 0, err
	}
	return image.ID, nil
}

// FindByID IDで画像レコードを検索
func (r *imageRepository) FindByID(id uint) (*models.Image, error) {
	var image models.Image
	if err := r.db.First(&image, id).Error; err != nil {
		return nil, err
	}
	return &image, nil
}

// Update 画像レコードを更新
func (r *imageRepository) Update(image *models.Image) error {
	return r.db.Save(image).Error
}

// Delete 画像レコードを削除
func (r *imageRepository) Delete(id uint) error {
	return r.db.Delete(&models.Image{}, id).Error
}

// ListPendingImages 変換待ちの画像一覧を取得
func (r *imageRepository) ListPendingImages(limit int) ([]models.Image, error) {
	var images []models.Image

	if err := r.db.Where("status = ?", "pending").
		Limit(limit).
		Order("created_at ASC").
		Find(&images).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return images, nil
}

// UpdateStatus 画像のステータスを更新
func (r *imageRepository) UpdateStatus(id uint, status string, webpPath string, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if webpPath != "" {
		updates["webp_path"] = webpPath
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	return r.db.Model(&models.Image{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// CountPendingImages 変換待ちの画像数をカウント
func (r *imageRepository) CountPendingImages() (int64, error) {
	var count int64
	if err := r.db.Model(&models.Image{}).
		Where("status = ?", "pending").
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// UpdateImageStats 画像の統計情報を更新
func (r *imageRepository) UpdateImageStats(id uint, webpPath string, originalSize int64, webpSize int64, compressionRatio float64, width int, height int) error {
	return r.db.Model(&models.Image{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"webp_path":         webpPath,
			"original_size":     originalSize,
			"webp_size":         webpSize,
			"compression_ratio": compressionRatio,
			"width":             width,
			"height":            height,
			"status":            "processed",
		}).Error
}
