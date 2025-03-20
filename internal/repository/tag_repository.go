package repository

import (
	"errors"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"

	"gorm.io/gorm"
)

// TagRepository タグに関するデータベース操作を行うインターフェース
type TagRepository interface {
	FindOrCreate(name string) (*models.Tag, error)
	List(search string, limit int) ([]models.Tag, error)
	FindByID(id uint) (*models.Tag, error)
	FindByName(name string) (*models.Tag, error)
	AttachTagsToWork(workID uint, tagIDs []uint) error
	DetachTagsFromWork(workID uint, tagIDs []uint) error
	GetTagsForWork(workID uint) ([]models.Tag, error)
}

// tagRepository TagRepositoryの実装
type tagRepository struct {
	db *gorm.DB
}

// NewTagRepository TagRepositoryを作成
func NewTagRepository(db *gorm.DB) TagRepository {
	return &tagRepository{db: db}
}

// FindOrCreate タグを検索または作成
func (r *tagRepository) FindOrCreate(name string) (*models.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, errors.New("タグ名は空にできません")
	}

	var tag models.Tag
	if err := r.db.Where("name = ?", name).First(&tag).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// タグが見つからない場合は新規作成
			tag.Name = name
			if err := r.db.Create(&tag).Error; err != nil {
				return nil, err
			}
			return &tag, nil
		}
		return nil, err
	}
	return &tag, nil
}

// List タグ一覧を取得
func (r *tagRepository) List(search string, limit int) ([]models.Tag, error) {
	var tags []models.Tag
	query := r.db.Model(&models.Tag{})

	if search != "" {
		query = query.Where("name LIKE ?", "%"+search+"%")
	}

	if err := query.
		Limit(limit).
		Order("name ASC").
		Find(&tags).Error; err != nil {
		return nil, err
	}

	return tags, nil
}

// FindByID IDでタグを検索
func (r *tagRepository) FindByID(id uint) (*models.Tag, error) {
	var tag models.Tag
	if err := r.db.First(&tag, id).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// FindByName 名前でタグを検索
func (r *tagRepository) FindByName(name string) (*models.Tag, error) {
	var tag models.Tag
	if err := r.db.Where("name = ?", name).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// AttachTagsToWork 作品にタグを関連付け
func (r *tagRepository) AttachTagsToWork(workID uint, tagIDs []uint) error {
	// 既存のタグ関連を取得
	var existingTagIDs []uint
	if err := r.db.Table("work_tags").
		Where("work_id = ?", workID).
		Pluck("tag_id", &existingTagIDs).Error; err != nil {
		return err
	}

	// 新しいタグだけを追加
	for _, tagID := range tagIDs {
		exists := false
		for _, existingID := range existingTagIDs {
			if tagID == existingID {
				exists = true
				break
			}
		}

		if !exists {
			if err := r.db.Exec("INSERT INTO work_tags (work_id, tag_id) VALUES (?, ?)", workID, tagID).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// DetachTagsFromWork 作品からタグの関連付けを解除
func (r *tagRepository) DetachTagsFromWork(workID uint, tagIDs []uint) error {
	if len(tagIDs) == 0 {
		return nil
	}
	// ちょっと怖い
	return r.db.Where("work_id = ? AND tag_id IN ?", workID, tagIDs).Delete(&WorkTag{}).Error
}

// GetTagsForWork 作品に関連付けられたタグを取得
func (r *tagRepository) GetTagsForWork(workID uint) ([]models.Tag, error) {
	var tags []models.Tag
	if err := r.db.Model(&models.Tag{}).
		Joins("JOIN work_tags ON tags.id = work_tags.tag_id").
		Where("work_tags.work_id = ?", workID).
		Find(&tags).Error; err != nil {
		return nil, err
	}
	return tags, nil
}

// WorkTag 作品とタグの中間テーブル用モデル
type WorkTag struct {
	WorkID uint `gorm:"primaryKey"`
	TagID  uint `gorm:"primaryKey"`
}

func (WorkTag) TableName() string {
	return "work_tags"
}
