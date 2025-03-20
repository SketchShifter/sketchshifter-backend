package repository

import (
	"errors"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"

	"gorm.io/gorm"
)

// CommentRepository コメントに関するデータベース操作を行うインターフェース
type CommentRepository interface {
	Create(comment *models.Comment) error
	FindByID(id uint) (*models.Comment, error)
	Update(comment *models.Comment) error
	Delete(id uint) error
	ListByWork(workID uint, page, limit int) ([]models.Comment, int64, error)
}

// commentRepository CommentRepositoryの実装
type commentRepository struct {
	db *gorm.DB
}

// NewCommentRepository CommentRepositoryを作成
func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

// Create 新しいコメントを作成
func (r *commentRepository) Create(comment *models.Comment) error {
	return r.db.Create(comment).Error
}

// FindByID IDでコメントを検索
func (r *commentRepository) FindByID(id uint) (*models.Comment, error) {
	var comment models.Comment
	if err := r.db.Preload("User").First(&comment, id).Error; err != nil {
		return nil, err
	}
	return &comment, nil
}

// Update コメントを更新
func (r *commentRepository) Update(comment *models.Comment) error {
	return r.db.Save(comment).Error
}

// Delete コメントを削除
func (r *commentRepository) Delete(id uint) error {
	return r.db.Delete(&models.Comment{}, id).Error
}

// ListByWork 作品のコメント一覧を取得
func (r *commentRepository) ListByWork(workID uint, page, limit int) ([]models.Comment, int64, error) {
	var comments []models.Comment
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&models.Comment{}).
		Where("work_id = ?", workID).
		Preload("User")

	// 合計数を取得
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// データを取得
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&comments).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, err
	}

	return comments, total, nil
}
