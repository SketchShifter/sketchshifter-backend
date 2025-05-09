package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// TaskService タスクに関するサービスインターフェース
type TaskService interface {
	Create(title, description string, projectID, userID uint) (*models.Task, error)
	GetByID(id uint, userID uint) (*models.Task, error)
	Update(id, userID uint, title, description string) (*models.Task, error)
	Delete(id, userID uint) error
	ListByProject(projectID, userID uint) ([]models.Task, error)
	AddWork(taskID, workID, userID uint) error
	RemoveWork(taskID, workID, userID uint) error
	GetWorks(taskID, userID uint, page, limit int) ([]models.Work, int64, int, error)
	UpdateOrders(taskIDs []uint, orderIndices []int, userID uint) error
}

// taskService TaskServiceの実装
type taskService struct {
	taskRepo    repository.TaskRepository
	projectRepo repository.ProjectRepository
	workRepo    repository.WorkRepository
}

// NewTaskService TaskServiceを作成
func NewTaskService(
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository,
	workRepo repository.WorkRepository,
) TaskService {
	return &taskService{
		taskRepo:    taskRepo,
		projectRepo: projectRepo,
		workRepo:    workRepo,
	}
}

// Create 新しいタスクを作成
func (s *taskService) Create(title, description string, projectID, userID uint) (*models.Task, error) {
	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// プロジェクトが存在するか確認
	_, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return nil, errors.New("プロジェクトが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(projectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("このプロジェクトにタスクを追加する権限がありません")
	}

	// 既存のタスク数を取得して順序を決定
	tasks, err := s.taskRepo.ListByProject(projectID)
	orderIndex := 0
	if err == nil {
		orderIndex = len(tasks)
	}

	// タスクを作成
	task := &models.Task{
		Title:       title,
		Description: description,
		ProjectID:   projectID,
		OrderIndex:  orderIndex,
	}

	// データベースに保存
	if err := s.taskRepo.Create(task); err != nil {
		return nil, fmt.Errorf("タスクの作成に失敗しました: %v", err)
	}

	return task, nil
}

// GetByID IDでタスクを取得
func (s *taskService) GetByID(id uint, userID uint) (*models.Task, error) {
	// タスクを取得
	task, err := s.taskRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// プロジェクトが存在するか確認
	_, err = s.projectRepo.FindByID(task.ProjectID)
	if err != nil {
		return nil, errors.New("プロジェクトが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("このタスクを閲覧する権限がありません")
	}

	return task, nil
}

// Update タスクを更新
func (s *taskService) Update(id, userID uint, title, description string) (*models.Task, error) {
	// タスクを取得
	task, err := s.taskRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("タスクが見つかりません")
	}

	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("このタスクを更新する権限がありません")
	}

	// フィールドを更新
	task.Title = title
	task.Description = description

	// データベースを更新
	if err := s.taskRepo.Update(task); err != nil {
		return nil, fmt.Errorf("タスクの更新に失敗しました: %v", err)
	}

	return task, nil
}

// Delete タスクを削除
func (s *taskService) Delete(id, userID uint) error {
	// タスクを取得
	task, err := s.taskRepo.FindByID(id)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのオーナーかどうか確認
	isOwner, err := s.projectRepo.IsOwner(task.ProjectID, userID)
	if err != nil || !isOwner {
		return errors.New("このタスクを削除する権限がありません")
	}

	// タスクを削除
	if err := s.taskRepo.Delete(id); err != nil {
		return fmt.Errorf("タスクの削除に失敗しました: %v", err)
	}

	return nil
}

// ListByProject プロジェクトのタスク一覧を取得
func (s *taskService) ListByProject(projectID, userID uint) ([]models.Task, error) {
	// プロジェクトが存在するか確認
	_, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return nil, errors.New("プロジェクトが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(projectID, userID)
	if err != nil || !isMember {
		return nil, errors.New("このプロジェクトのタスク一覧を閲覧する権限がありません")
	}

	// タスク一覧を取得
	return s.taskRepo.ListByProject(projectID)
}

// AddWork 作品をタスクに追加
func (s *taskService) AddWork(taskID, workID, userID uint) error {
	// タスクを取得
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// 作品を取得
	work, err := s.workRepo.FindByID(workID)
	if err != nil {
		return errors.New("作品が見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return errors.New("このタスクに作品を追加する権限がありません")
	}

	// 作品の所有者かどうか確認
	if work.UserID != userID {
		// オーナーは他のメンバーの作品も追加できる
		isOwner, err := s.projectRepo.IsOwner(task.ProjectID, userID)
		if err != nil || !isOwner {
			return errors.New("他のユーザーの作品をタスクに追加する権限がありません")
		}
	}

	// 作品をタスクに追加
	return s.taskRepo.AddWork(taskID, workID)
}

// RemoveWork 作品をタスクから削除
func (s *taskService) RemoveWork(taskID, workID, userID uint) error {
	// タスクを取得
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	// 作品を取得
	work, err := s.workRepo.FindByID(workID)
	if err != nil {
		return errors.New("作品が見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return errors.New("このタスクから作品を削除する権限がありません")
	}

	// 作品の所有者かどうか確認
	if work.UserID != userID {
		// オーナーは他のメンバーの作品も削除できる
		isOwner, err := s.projectRepo.IsOwner(task.ProjectID, userID)
		if err != nil || !isOwner {
			return errors.New("他のユーザーの作品をタスクから削除する権限がありません")
		}
	}

	// 作品をタスクから削除
	return s.taskRepo.RemoveWork(taskID, workID)
}

// GetWorks タスクの作品一覧を取得
func (s *taskService) GetWorks(taskID, userID uint, page, limit int) ([]models.Work, int64, int, error) {
	// タスクを取得
	task, err := s.taskRepo.FindByID(taskID)
	if err != nil {
		return nil, 0, 0, errors.New("タスクが見つかりません")
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
	if err != nil || !isMember {
		return nil, 0, 0, errors.New("このタスクの作品一覧を閲覧する権限がありません")
	}

	// 作品一覧を取得
	works, total, err := s.taskRepo.GetWorks(taskID, page, limit)
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

// UpdateOrders タスクの表示順序を更新
func (s *taskService) UpdateOrders(taskIDs []uint, orderIndices []int, userID uint) error {
	if len(taskIDs) == 0 || len(taskIDs) != len(orderIndices) {
		return errors.New("無効なタスクIDまたは順序インデックスです")
	}

	// 最初のタスクからプロジェクトIDを取得
	task, err := s.taskRepo.FindByID(taskIDs[0])
	if err != nil {
		return errors.New("タスクが見つかりません")
	}

	projectID := task.ProjectID

	// 全てのタスクが同じプロジェクトに属しているか確認
	for _, taskID := range taskIDs {
		task, err := s.taskRepo.FindByID(taskID)
		if err != nil {
			return errors.New("タスクが見つかりません")
		}

		if task.ProjectID != projectID {
			return errors.New("異なるプロジェクトのタスクの順序を一度に更新することはできません")
		}
	}

	// ユーザーがプロジェクトのメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(projectID, userID)
	if err != nil || !isMember {
		return errors.New("このプロジェクトのタスク順序を更新する権限がありません")
	}

	// タスクの順序を更新
	return s.taskRepo.UpdateOrders(taskIDs, orderIndices)
}
