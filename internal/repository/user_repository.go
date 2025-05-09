package repository

import (
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
