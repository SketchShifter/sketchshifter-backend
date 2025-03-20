package services

import (
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// TagService タグに関するサービスインターフェース
type TagService interface {
	List(search string, limit int) ([]models.Tag, error)
}

// tagService TagServiceの実装
type tagService struct {
	tagRepo repository.TagRepository
}

// NewTagService TagServiceを作成
func NewTagService(tagRepo repository.TagRepository) TagService {
	return &tagService{
		tagRepo: tagRepo,
	}
}

// List タグ一覧を取得
func (s *tagService) List(search string, limit int) ([]models.Tag, error) {
	return s.tagRepo.List(search, limit)
}
