package services

import (
	"errors"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// CommentService コメントに関するサービスインターフェース
type CommentService interface {
	Create(content string, workID uint, userID uint) (*models.Comment, error)
	GetByID(id uint) (*models.Comment, error)
	Update(id, userID uint, content string) (*models.Comment, error)
	Delete(id, userID uint) error
	ListByWork(workID uint, page, limit int) ([]models.Comment, int64, int, error)
}

// commentService CommentServiceの実装
type commentService struct {
	commentRepo repository.CommentRepository
	workRepo    repository.WorkRepository
}

// NewCommentService CommentServiceを作成
func NewCommentService(commentRepo repository.CommentRepository, workRepo repository.WorkRepository) CommentService {
	return &commentService{
		commentRepo: commentRepo,
		workRepo:    workRepo,
	}
}

// Create 新しいコメントを作成
func (s *commentService) Create(content string, workID uint, userID uint) (*models.Comment, error) {
	// コンテンツのバリデーション
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("コメント内容は必須です")
	}

	// 作品が存在するか確認
	_, err := s.workRepo.FindByID(workID)
	if err != nil {
		return nil, errors.New("作品が見つかりません")
	}

	// 新しいコメントを作成
	comment := &models.Comment{
		Content: content,
		WorkID:  workID,
		UserID:  userID,
	}

	// データベースに保存
	if err := s.commentRepo.Create(comment); err != nil {
		return nil, err
	}

	return s.GetByID(comment.ID)
}

// GetByID IDでコメントを取得
func (s *commentService) GetByID(id uint) (*models.Comment, error) {
	return s.commentRepo.FindByID(id)
}

// Update コメントを更新
func (s *commentService) Update(id, userID uint, content string) (*models.Comment, error) {
	// コンテンツのバリデーション
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("コメント内容は必須です")
	}

	// コメントを取得
	comment, err := s.commentRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("コメントが見つかりません")
	}

	// 権限チェック
	if comment.UserID != userID {
		return nil, errors.New("このコメントを更新する権限がありません")
	}

	// コンテンツを更新
	comment.Content = content

	// データベースを更新
	if err := s.commentRepo.Update(comment); err != nil {
		return nil, err
	}

	return s.GetByID(id)
}

// Delete コメントを削除
func (s *commentService) Delete(id, userID uint) error {
	// コメントを取得
	comment, err := s.commentRepo.FindByID(id)
	if err != nil {
		return errors.New("コメントが見つかりません")
	}

	// 権限チェック
	if comment.UserID != userID {
		return errors.New("このコメントを削除する権限がありません")
	}

	// データベースから削除
	return s.commentRepo.Delete(id)
}

// ListByWork 作品のコメント一覧を取得
func (s *commentService) ListByWork(workID uint, page, limit int) ([]models.Comment, int64, int, error) {
	// 作品が存在するか確認
	_, err := s.workRepo.FindByID(workID)
	if err != nil {
		return nil, 0, 0, errors.New("作品が見つかりません")
	}

	// コメント一覧を取得
	comments, total, err := s.commentRepo.ListByWork(workID, page, limit)
	if err != nil {
		return nil, 0, 0, err
	}

	// 総ページ数を計算
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}

	return comments, total, pages, nil
}
