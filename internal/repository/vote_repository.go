package repository

import (
	"errors"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"gorm.io/gorm"
)

// VoteRepository 投票に関するデータベース操作を行うインターフェース
type VoteRepository interface {
	Create(vote *models.Vote) error
	FindByID(id uint) (*models.Vote, error)
	Update(vote *models.Vote) error
	Delete(id uint) error
	ListByTask(taskID uint) ([]models.Vote, error)
	CreateOption(option *models.VoteOption) error
	FindOptionByID(id uint) (*models.VoteOption, error)
	DeleteOption(id uint) error
	GetOptions(voteID uint) ([]models.VoteOption, error)
	AddResponse(response *models.VoteResponse) error
	RemoveResponse(voteID, optionID, userID uint) error
	GetUserResponses(voteID, userID uint) ([]models.VoteResponse, error)
	GetOptionVoteCounts(voteID uint) (map[uint]int64, error)
	CloseVote(voteID uint) error
}

// voteRepository VoteRepositoryの実装
type voteRepository struct {
	db *gorm.DB
}

// NewVoteRepository VoteRepositoryを作成
func NewVoteRepository(db *gorm.DB) VoteRepository {
	return &voteRepository{db: db}
}

// Create 新しい投票を作成
func (r *voteRepository) Create(vote *models.Vote) error {
	return r.db.Create(vote).Error
}

// FindByID IDで投票を検索
func (r *voteRepository) FindByID(id uint) (*models.Vote, error) {
	var vote models.Vote
	if err := r.db.Preload("Creator").Preload("Options").First(&vote, id).Error; err != nil {
		return nil, err
	}

	// 各オプションの投票数を取得
	voteCounts, err := r.GetOptionVoteCounts(id)
	if err != nil {
		return nil, err
	}

	for i := range vote.Options {
		vote.Options[i].VoteCount = voteCounts[vote.Options[i].ID]
	}

	return &vote, nil
}

// Update 投票情報を更新
func (r *voteRepository) Update(vote *models.Vote) error {
	return r.db.Save(vote).Error
}

// Delete 投票を削除
func (r *voteRepository) Delete(id uint) error {
	return r.db.Delete(&models.Vote{}, id).Error
}

// ListByTask タスクの投票一覧を取得
func (r *voteRepository) ListByTask(taskID uint) ([]models.Vote, error) {
	var votes []models.Vote

	if err := r.db.Where("task_id = ?", taskID).
		Preload("Creator").
		Preload("Options").
		Order("created_at DESC").
		Find(&votes).Error; err != nil {
		return nil, err
	}

	// 各投票の各オプションの投票数を取得
	for i := range votes {
		voteCounts, err := r.GetOptionVoteCounts(votes[i].ID)
		if err != nil {
			return nil, err
		}

		for j := range votes[i].Options {
			votes[i].Options[j].VoteCount = voteCounts[votes[i].Options[j].ID]
		}
	}

	return votes, nil
}

// CreateOption 新しい投票オプションを作成
func (r *voteRepository) CreateOption(option *models.VoteOption) error {
	return r.db.Create(option).Error
}

// FindOptionByID IDで投票オプションを検索
func (r *voteRepository) FindOptionByID(id uint) (*models.VoteOption, error) {
	var option models.VoteOption
	if err := r.db.Preload("Work").First(&option, id).Error; err != nil {
		return nil, err
	}

	// 投票数を取得
	var count int64
	if err := r.db.Model(&models.VoteResponse{}).
		Where("option_id = ?", id).
		Count(&count).Error; err != nil {
		return nil, err
	}

	option.VoteCount = count

	return &option, nil
}

// DeleteOption 投票オプションを削除
func (r *voteRepository) DeleteOption(id uint) error {
	return r.db.Delete(&models.VoteOption{}, id).Error
}

// GetOptions 投票のオプション一覧を取得
func (r *voteRepository) GetOptions(voteID uint) ([]models.VoteOption, error) {
	var options []models.VoteOption

	if err := r.db.Where("vote_id = ?", voteID).
		Preload("Work").
		Order("created_at ASC").
		Find(&options).Error; err != nil {
		return nil, err
	}

	// 各オプションの投票数を取得
	voteCounts, err := r.GetOptionVoteCounts(voteID)
	if err != nil {
		return nil, err
	}

	for i := range options {
		options[i].VoteCount = voteCounts[options[i].ID]
	}

	return options, nil
}

// AddResponse 投票回答を追加
func (r *voteRepository) AddResponse(response *models.VoteResponse) error {
	// 投票が有効かどうか確認
	var vote models.Vote
	if err := r.db.Select("is_active").First(&vote, response.VoteID).Error; err != nil {
		return err
	}

	if !vote.IsActive {
		return errors.New("この投票は既に終了しています")
	}

	// 回答を追加
	return r.db.Create(response).Error
}

// RemoveResponse 投票回答を削除
func (r *voteRepository) RemoveResponse(voteID, optionID, userID uint) error {
	return r.db.Where("vote_id = ? AND option_id = ? AND user_id = ?", voteID, optionID, userID).
		Delete(&models.VoteResponse{}).Error
}

// GetUserResponses ユーザーの投票回答を取得
func (r *voteRepository) GetUserResponses(voteID, userID uint) ([]models.VoteResponse, error) {
	var responses []models.VoteResponse

	if err := r.db.Where("vote_id = ? AND user_id = ?", voteID, userID).
		Find(&responses).Error; err != nil {
		return nil, err
	}

	return responses, nil
}

// GetOptionVoteCounts オプションごとの投票数を取得
func (r *voteRepository) GetOptionVoteCounts(voteID uint) (map[uint]int64, error) {
	type Result struct {
		OptionID uint
		Count    int64
	}

	var results []Result
	err := r.db.Model(&models.VoteResponse{}).
		Select("option_id, count(*) as count").
		Where("vote_id = ?", voteID).
		Group("option_id").
		Find(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[uint]int64)
	for _, result := range results {
		counts[result.OptionID] = result.Count
	}

	return counts, nil
}

// CloseVote 投票を終了
func (r *voteRepository) CloseVote(voteID uint) error {
	now := time.Now()
	return r.db.Model(&models.Vote{}).
		Where("id = ?", voteID).
		Updates(map[string]interface{}{
			"is_active": false,
			"closed_at": now,
		}).Error
}
