package repository

import (
	"errors"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"

	"gorm.io/gorm"
)

// UserRepository ユーザーに関するデータベース操作を行うインターフェース
type UserRepository interface {
	Create(user *models.User) error
	FindByID(id uint) (*models.User, error)
	FindByEmail(email string) (*models.User, error)
	Update(user *models.User) error
	Delete(id uint) error
	GetUserFavorites(userID uint, page, limit int) ([]models.Work, int64, error)
}

// userRepository UserRepositoryの実装
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository UserRepositoryを作成
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 新しいユーザーを作成
func (r *userRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// FindByID IDでユーザーを検索
func (r *userRepository) FindByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail メールアドレスでユーザーを検索
func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// Update ユーザー情報を更新
func (r *userRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// Delete ユーザーを削除
func (r *userRepository) Delete(id uint) error {
	return r.db.Delete(&models.User{}, id).Error
}

// GetUserFavorites ユーザーのお気に入り作品を取得
func (r *userRepository) GetUserFavorites(userID uint, page, limit int) ([]models.Work, int64, error) {
	var works []models.Work
	var total int64

	offset := (page - 1) * limit

	query := r.db.Table("works").
		Joins("INNER JOIN likes ON works.id = likes.work_id").
		Where("likes.user_id = ?", userID).
		Preload("User").
		Preload("Tags")

	// 合計数を取得
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// データを取得
	if err := query.
		Offset(offset).
		Limit(limit).
		Order("likes.created_at DESC").
		Find(&works).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, err
	}

	// 各作品のいいね数とコメント数を取得
	for i := range works {
		r.db.Model(&models.Like{}).Where("work_id = ?", works[i].ID).Count(&works[i].LikesCount)
		r.db.Model(&models.Comment{}).Where("work_id = ?", works[i].ID).Count(&works[i].CommentsCount)
	}

	return works, total, nil
}
