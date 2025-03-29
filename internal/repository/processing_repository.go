package repository

import (
	"github.com/SketchShifter/sketchshifter_backend/internal/models"

	"gorm.io/gorm"
)

// ProcessingRepository Processing作品に関するデータベース操作を行うインターフェース
type ProcessingRepository interface {
	Create(processing *models.ProcessingWork) (uint, error)
	FindByID(id uint) (*models.ProcessingWork, error)
	Update(processing *models.ProcessingWork) error
	Delete(id uint) error
	FindByWorkID(workID uint) (*models.ProcessingWork, error)
	UpdateStatus(id uint, status string, jsPath string, errorMessage string) error
	GetPDEContent(id uint) (string, error)
	// バッチ処理用の追加メソッド
	ListPendingProcessings(limit int) ([]models.ProcessingWork, error)
	CountPendingProcessings() (int64, error)
}

// processingRepository ProcessingRepositoryの実装
type processingRepository struct {
	db *gorm.DB
}

// NewProcessingRepository ProcessingRepositoryを作成
func NewProcessingRepository(db *gorm.DB) ProcessingRepository {
	return &processingRepository{db: db}
}

// Create 新しいProcessing作品レコードを作成
func (r *processingRepository) Create(processing *models.ProcessingWork) (uint, error) {
	if err := r.db.Create(processing).Error; err != nil {
		return 0, err
	}
	return processing.ID, nil
}

// FindByID IDでProcessing作品レコードを検索
func (r *processingRepository) FindByID(id uint) (*models.ProcessingWork, error) {
	var processing models.ProcessingWork
	if err := r.db.First(&processing, id).Error; err != nil {
		return nil, err
	}
	return &processing, nil
}

// Update Processing作品レコードを更新
func (r *processingRepository) Update(processing *models.ProcessingWork) error {
	return r.db.Save(processing).Error
}

// Delete Processing作品レコードを削除
func (r *processingRepository) Delete(id uint) error {
	return r.db.Delete(&models.ProcessingWork{}, id).Error
}

// FindByWorkID 作品IDからProcessing作品レコードを検索
func (r *processingRepository) FindByWorkID(workID uint) (*models.ProcessingWork, error) {
	var processing models.ProcessingWork
	if err := r.db.Where("work_id = ?", workID).First(&processing).Error; err != nil {
		return nil, err
	}
	return &processing, nil
}

// UpdateStatus Processing作品のステータスを更新
func (r *processingRepository) UpdateStatus(id uint, status string, jsPath string, errorMessage string) error {
	updates := map[string]interface{}{
		"status": status,
	}

	if jsPath != "" {
		updates["js_path"] = jsPath
	}

	if errorMessage != "" {
		updates["error_message"] = errorMessage
	}

	return r.db.Model(&models.ProcessingWork{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetPDEContent PDEコンテンツを取得
func (r *processingRepository) GetPDEContent(id uint) (string, error) {
	var processing models.ProcessingWork
	if err := r.db.Select("pde_content").First(&processing, id).Error; err != nil {
		return "", err
	}
	return processing.PDEContent, nil
}

// ListPendingProcessings 未処理のProcessing作品リストを取得
func (r *processingRepository) ListPendingProcessings(limit int) ([]models.ProcessingWork, error) {
	var processings []models.ProcessingWork

	if err := r.db.Where("status = ?", "pending").
		Limit(limit).
		Order("created_at ASC").
		Find(&processings).Error; err != nil {
		return nil, err
	}

	return processings, nil
}

// CountPendingProcessings 未処理のProcessing作品数をカウント
func (r *processingRepository) CountPendingProcessings() (int64, error) {
	var count int64
	if err := r.db.Model(&models.ProcessingWork{}).
		Where("status = ?", "pending").
		Count(&count).Error; err != nil {
		return 0, err
	}

	return count, nil
}
