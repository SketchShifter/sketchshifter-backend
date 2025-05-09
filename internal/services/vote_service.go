package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// VoteService 投票に関するサービスインターフェース
type VoteService interface {
	Create(title, description string, taskID uint, multiSelect bool, userID uint) (*models.Vote, error)
	GetByID(id, userID uint) (*models.Vote, error)
	Update(id, userID uint, title, description string, multiSelect bool) (*models.Vote, error)
	Delete(id, userID uint) error
	ListByTask(taskID, userID uint) ([]models.Vote, error)
	AddOption(voteID, userID uint, optionText string, workID *uint) (*models.VoteOption, error)
	DeleteOption(optionID, userID uint) error
	Vote(voteID, optionID, userID uint) error
	RemoveVote(voteID, optionID, userID uint) error
	GetUserVotes(voteID, userID uint) ([]models.VoteResponse, error)
	CloseVote(voteID, userID uint) error
}

// voteService VoteServiceの実装
type voteService struct {
	voteRepo    repository.VoteRepository
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	workRepo    repository.WorkRepository
}

// NewVoteService VoteServiceを作成
func NewVoteService(
	voteRepo repository.VoteRepository,
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	workRepo repository.WorkRepository,
) VoteService {
	return &voteService{
		voteRepo:    voteRepo,
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		workRepo:    workRepo,
	}
}

// Create 新しい投票を作成
func (s *voteService) Create(title, description string, taskID uint, multiSelect bool, userID uint) (*models.Vote, error) {
	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("このタスクに投票を追加する権限がありません")
	}

	// 投票を作成
	vote := &models.Vote{
		Title:       title,
		Description: description,
		TaskID:      taskID,
		MultiSelect: multiSelect,
		IsActive:    true,
		CreatedBy:   userID,
	}

	// データベースに保存
	if err := s.voteRepo.Create(vote); err != nil {
		return nil, fmt.Errorf("投票の作成に失敗しました: %v", err)
	}

	return s.GetByID(vote.ID, userID)
}

// GetByID IDで投票を取得
func (s *voteService) GetByID(id, userID uint) (*models.Vote, error) {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("投票が見つかりません")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("この投票を閲覧する権限がありません")
	}

	return vote, nil
}

// Update 投票を更新
func (s *voteService) Update(id, userID uint, title, description string, multiSelect bool) (*models.Vote, error) {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("投票が見つかりません")
	}

	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// 投票が作成者またはプロジェクトのオーナーかどうか確認
	if vote.CreatedBy != userID {
		isOwner, err := s.projectRepo.IsOwner(task.ProjectID, userID)
		if err != nil || !isOwner {
			return nil, errors.New("この投票を更新する権限がありません")
		}
	}

	// すでに投票が行われている場合は、マルチセレクト設定を変更できない
	if vote.MultiSelect != multiSelect {
		// 投票がすでに行われているか確認
		voteOptions, err := s.voteRepo.GetOptions(id)
		if err != nil {
			return nil, fmt.Errorf("投票オプションの取得に失敗しました: %v", err)
		}

		for _, option := range voteOptions {
			if option.VoteCount > 0 {
				return nil, errors.New("投票が既に行われているため、マルチセレクト設定を変更できません")
			}
		}
	}

	// フィールドを更新
	vote.Title = title
	vote.Description = description
	vote.MultiSelect = multiSelect

	// データベースを更新
	if err := s.voteRepo.Update(vote); err != nil {
		return nil, fmt.Errorf("投票の更新に失敗しました: %v", err)
	}

	return s.GetByID(id, userID)
}

// Delete 投票を削除
func (s *voteService) Delete(id, userID uint) error {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(id)
	if err != nil {
		return errors.New("投票が見つかりません")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// 投票が作成者またはプロジェクトのオーナーかどうか確認
	if vote.CreatedBy != userID {
		isOwner, err := s.projectRepo.IsOwner(task.ProjectID, userID)
		if err != nil || !isOwner {
			return errors.New("この投票を削除する権限がありません")
		}
	}

	// 投票を削除
	if err := s.voteRepo.Delete(id); err != nil {
		return fmt.Errorf("投票の削除に失敗しました: %v", err)
	}

	return nil
}

// ListByTask タスクの投票一覧を取得
func (s *voteService) ListByTask(taskID, userID uint) ([]models.Vote, error) {
	// タスクを取得
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("このタスクの投票一覧を閲覧する権限がありません")
	}

	// 投票一覧を取得
	return s.voteRepo.ListByTask(taskID)
}

// AddOption 投票オプションを追加
func (s *voteService) AddOption(voteID, userID uint, optionText string, workID *uint) (*models.VoteOption, error) {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(voteID)
	if err != nil {
		return nil, errors.New("投票が見つかりません")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("この投票にオプションを追加する権限がありません")
	}

	// オプションテキストのバリデーション
	if strings.TrimSpace(optionText) == "" {
		return nil, errors.New("オプションテキストは必須です")
	}

	// 作品IDがある場合は、作品が存在するか確認
	if workID != nil {
		_, err := s.workRepo.FindByID(*workID)
		if err != nil {
			return nil, errors.New("作品が見つかりません")
		}
	}

	// オプションを作成
	option := &models.VoteOption{
		VoteID:     voteID,
		OptionText: optionText,
		WorkID:     workID,
	}

	// データベースに保存
	if err := s.voteRepo.CreateOption(option); err != nil {
		return nil, fmt.Errorf("投票オプションの作成に失敗しました: %v", err)
	}

	return s.voteRepo.FindOptionByID(option.ID)
}

// DeleteOption 投票オプションを削除
func (s *voteService) DeleteOption(optionID, userID uint) error {
	// オプションを取得
	option, err := s.voteRepo.FindOptionByID(optionID)
	if err != nil {
		return errors.New("投票オプションが見つかりません")
	}

	// 投票を取得
	vote, err := s.voteRepo.FindByID(option.VoteID)
	if err != nil {
		return errors.New("投票が見つかりません")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// 投票が作成者またはプロジェクトのオーナーかどうか確認
	if vote.CreatedBy != userID {
		isOwner, err := s.projectRepo.IsOwner(task.ProjectID, userID)
		if err != nil || !isOwner {
			return errors.New("この投票オプションを削除する権限がありません")
		}
	}

	// 投票がすでに行われている場合は削除できない
	if option.VoteCount > 0 {
		return errors.New("投票が既に行われているため、このオプションを削除できません")
	}

	// オプションを削除
	if err := s.voteRepo.DeleteOption(optionID); err != nil {
		return fmt.Errorf("投票オプションの削除に失敗しました: %v", err)
	}

	return nil
}

// Vote 投票する
func (s *voteService) Vote(voteID, optionID, userID uint) error {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(voteID)
	if err != nil {
		return errors.New("投票が見つかりません")
	}

	// 投票が有効かどうか確認
	if !vote.IsActive {
		return errors.New("この投票は既に終了しています")
	}

	// オプションを取得
	option, err := s.voteRepo.FindOptionByID(optionID)
	if err != nil {
		return errors.New("投票オプションが見つかりません")
	}

	// オプションが投票に属しているか確認
	if option.VoteID != voteID {
		return errors.New("このオプションはこの投票に属していません")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return errors.New("この投票に参加する権限がありません")
	}

	// マルチセレクトでない場合は、既存の投票を取得
	if !vote.MultiSelect {
		// ユーザーの投票を取得
		responses, err := s.voteRepo.GetUserResponses(voteID, userID)
		if err != nil {
			return fmt.Errorf("投票情報の取得に失敗しました: %v", err)
		}

		// すでに同じオプションに投票している場合は何もしない
		for _, response := range responses {
			if response.OptionID == optionID {
				return nil
			}
		}

		// 他のオプションに投票している場合は削除
		for _, response := range responses {
			if err := s.voteRepo.RemoveResponse(voteID, response.OptionID, userID); err != nil {
				return fmt.Errorf("既存の投票の削除に失敗しました: %v", err)
			}
		}
	} else {
		// マルチセレクトの場合は、すでに同じオプションに投票していないか確認
		responses, err := s.voteRepo.GetUserResponses(voteID, userID)
		if err != nil {
			return fmt.Errorf("投票情報の取得に失敗しました: %v", err)
		}

		// すでに同じオプションに投票している場合は何もしない
		for _, response := range responses {
			if response.OptionID == optionID {
				return nil
			}
		}
	}

	// 投票を追加
	response := &models.VoteResponse{
		VoteID:   voteID,
		OptionID: optionID,
		UserID:   userID,
	}

	return s.voteRepo.AddResponse(response)
}

// RemoveVote 投票を削除
func (s *voteService) RemoveVote(voteID, optionID, userID uint) error {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(voteID)
	if err != nil {
		return errors.New("投票が見つかりません")
	}

	// 投票が有効かどうか確認
	if !vote.IsActive {
		return errors.New("この投票は既に終了しています")
	}

	// オプションを取得
	option, err := s.voteRepo.FindOptionByID(optionID)
	if err != nil {
		return errors.New("投票オプションが見つかりません")
	}

	// オプションが投票に属しているか確認
	if option.VoteID != voteID {
		return errors.New("このオプションはこの投票に属していません")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return errors.New("この投票を削除する権限がありません")
	}

	// 投票を削除
	return s.voteRepo.RemoveResponse(voteID, optionID, userID)
}

// GetUserVotes ユーザーの投票を取得
func (s *voteService) GetUserVotes(voteID, userID uint) ([]models.VoteResponse, error) {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(voteID)
	if err != nil {
		return nil, errors.New("投票が見つかりません")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("この投票を閲覧する権限がありません")
	}

	// ユーザーの投票を取得
	return s.voteRepo.GetUserResponses(voteID, userID)
}

// CloseVote 投票を終了
func (s *voteService) CloseVote(voteID, userID uint) error {
	// 投票を取得
	vote, err := s.voteRepo.FindByID(voteID)
	if err != nil {
		return errors.New("投票が見つかりません")
	}

	// 投票が既に終了しているか確認
	if !vote.IsActive {
		return errors.New("この投票は既に終了しています")
	}

	// タスクを取得
	task, err := s.taskRepo.FindByID(vote.TaskID)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// 投票が作成者またはプロジェクトのオーナーかどうか確認
	if vote.CreatedBy != userID {
		isOwner, err := s.projectRepo.IsOwner(task.ProjectID, userID)
		if err != nil || !isOwner {
			return errors.New("この投票を終了する権限がありません")
		}
	}

	// 投票を終了
	return s.voteRepo.CloseVote(voteID)
}
