package repository

import (
	"errors"
	"fmt"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"

	"gorm.io/gorm"
)

// WorkRepository 作品に関するデータベース操作を行うインターフェース
type WorkRepository interface {
	Create(work *models.Work) error
	FindByID(id uint) (*models.Work, error)
	Update(work *models.Work) error
	Delete(id uint) error
	List(page, limit int, search, tag string, userID *uint, sort string) ([]models.Work, int64, error)
	IncrementViews(id uint) error
	AddLike(userID, workID uint) error
	RemoveLike(userID, workID uint) error
	GetLikesCount(workID uint) (int, error)
	HasLiked(userID, workID uint) (bool, error)
	ListByUser(userID uint, page, limit int) ([]models.Work, int64, error)
}

// workRepository WorkRepositoryの実装
type workRepository struct {
	db *gorm.DB
}

// NewWorkRepository WorkRepositoryを作成
func NewWorkRepository(db *gorm.DB) WorkRepository {
	return &workRepository{db: db}
}

// Create 新しい作品を作成
func (r *workRepository) Create(work *models.Work) error {
	return r.db.Create(work).Error
}

// FindByID IDで作品を検索
func (r *workRepository) FindByID(id uint) (*models.Work, error) {
	var work models.Work
	if err := r.db.Preload("User").Preload("Tags").First(&work, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("作品が見つかりません: ID=%d", id)
		}
		return nil, err
	}

	// いいね数とコメント数を取得
	r.db.Model(&models.Like{}).Where("work_id = ?", work.ID).Count(&work.LikesCount)
	r.db.Model(&models.Comment{}).Where("work_id = ?", work.ID).Count(&work.CommentsCount)

	return &work, nil
}

// Update 作品情報を更新
func (r *workRepository) Update(work *models.Work) error {
	return r.db.Save(work).Error
}

// Delete 作品を削除
func (r *workRepository) Delete(id uint) error {
	return r.db.Delete(&models.Work{}, id).Error
}

// IncrementViews 閲覧数を増加
func (r *workRepository) IncrementViews(id uint) error {
	return r.db.Model(&models.Work{}).Where("id = ?", id).Update("views", gorm.Expr("views + 1")).Error
}

// List 作品一覧を取得
func (r *workRepository) List(page, limit int, search, tag string, userID *uint, sort string) ([]models.Work, int64, error) {
	var works []models.Work
	var total int64

	offset := (page - 1) * limit

	// クエリビルダーを初期化
	query := r.db.Model(&models.Work{}).Preload("User").Preload("Tags")

	// 検索条件を適用
	if search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// タグでフィルタリング
	if tag != "" {
		query = query.Joins("JOIN work_tags ON works.id = work_tags.work_id").
			Joins("JOIN tags ON work_tags.tag_id = tags.id").
			Where("tags.name = ?", tag)
	}

	// ユーザーでフィルタリング
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	// 合計数を取得
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ソート順を適用
	switch sort {
	case "popular":
		// 人気順（閲覧数とお気に入り数の組み合わせでソート）
		query = query.Order("views DESC")
	case "views":
		// 閲覧数順
		query = query.Order("views DESC")
	default:
		// 新着順
		query = query.Order("created_at DESC")
	}

	// データを取得
	if err := query.
		Offset(offset).
		Limit(limit).
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

// AddLike いいねを追加
func (r *workRepository) AddLike(userID, workID uint) error {
	// 作品の存在確認
	var work models.Work
	if err := r.db.First(&work, workID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("作品が見つかりません: ID=%d", workID)
		}
		return err
	}

	// すでにいいねしているか確認
	var count int64
	r.db.Model(&models.Like{}).Where("user_id = ? AND work_id = ?", userID, workID).Count(&count)
	if count > 0 {
		return errors.New("既にいいねしています")
	}

	// いいねを作成
	like := models.Like{
		UserID: userID,
		WorkID: workID,
	}
	return r.db.Create(&like).Error
}

// RemoveLike いいねを削除
func (r *workRepository) RemoveLike(userID, workID uint) error {
	// 作品の存在確認
	var work models.Work
	if err := r.db.First(&work, workID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("作品が見つかりません: ID=%d", workID)
		}
		return err
	}

	// いいねを削除
	result := r.db.Where("user_id = ? AND work_id = ?", userID, workID).Delete(&models.Like{})
	if result.RowsAffected == 0 {
		return errors.New("いいねが見つかりません")
	}

	return result.Error
}

// GetLikesCount いいね数を取得
func (r *workRepository) GetLikesCount(workID uint) (int, error) {
	var count int64
	if err := r.db.Model(&models.Like{}).Where("work_id = ?", workID).Count(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

// HasLiked ユーザーがいいねしているか確認
func (r *workRepository) HasLiked(userID, workID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.Like{}).Where("user_id = ? AND work_id = ?", userID, workID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// ListByUser ユーザーの作品一覧を取得
func (r *workRepository) ListByUser(userID uint, page, limit int) ([]models.Work, int64, error) {
	var works []models.Work
	var total int64

	offset := (page - 1) * limit

	// ユーザーの存在確認
	var userCount int64
	if err := r.db.Model(&models.User{}).Where("id = ?", userID).Count(&userCount).Error; err != nil {
		return nil, 0, err
	}
	if userCount == 0 {
		return nil, 0, fmt.Errorf("ユーザーが見つかりません: ID=%d", userID)
	}

	// クエリを作成
	query := r.db.Model(&models.Work{}).
		Where("user_id = ?", userID).
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
		Order("created_at DESC").
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
