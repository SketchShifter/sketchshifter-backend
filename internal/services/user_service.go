package services

import (
	"errors"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// UserService ユーザーに関するサービスインターフェース
type UserService interface {
	GetByID(id uint) (*models.User, error)
	GetUserWorks(userID uint, page, limit int) ([]models.Work, int64, int, error)
	UpdateProfile(userID uint, name, nickname, bio string) (*models.User, error)
}

// userService UserServiceの実装
type userService struct {
	userRepo repository.UserRepository
	workRepo repository.WorkRepository
}

// NewUserService UserServiceを作成
func NewUserService(userRepo repository.UserRepository, workRepo repository.WorkRepository) UserService {
	return &userService{
		userRepo: userRepo,
		workRepo: workRepo,
	}
}

// GetByID IDでユーザーを取得
func (s *userService) GetByID(id uint) (*models.User, error) {
	return s.userRepo.FindByID(id)
}

// GetUserWorks ユーザーの作品一覧を取得
func (s *userService) GetUserWorks(userID uint, page, limit int) ([]models.Work, int64, int, error) {
	// ユーザーが存在するか確認
	_, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, 0, 0, errors.New("ユーザーが見つかりません")
	}

	// 作品一覧を取得
	works, total, err := s.workRepo.ListByUser(userID, page, limit)
	if err != nil {
		return nil, 0, 0, err
	}

	// 総ページ数を計算
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}

	return works, total, pages, nil
}

// UpdateProfile ユーザープロフィールを更新
func (s *userService) UpdateProfile(userID uint, name, nickname, bio string) (*models.User, error) {
	// ユーザーを取得
	user, err := s.userRepo.FindByID(userID)
	if err != nil {
		return nil, err
	}

	// フィールドを更新（空でない場合のみ）
	if strings.TrimSpace(name) != "" {
		user.Name = name
	}
	if strings.TrimSpace(nickname) != "" {
		user.Nickname = nickname
	}

	// bioはnullableなので空文字でも更新する
	user.Bio = bio

	// データベースを更新
	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user, nil
}
