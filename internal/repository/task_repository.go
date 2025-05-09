package repository

import (
	"errors"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"gorm.io/gorm"
)

// TaskRepository タスクに関するデータベース操作を行うインターフェース
type TaskRepository interface {
	Create(task *models.Task) error
	FindByID(id uint) (*models.Task, error)
	Update(task *models.Task) error
	Delete(id uint) error
	ListByProject(projectID uint) ([]models.Task, error)
	AddWork(taskID, workID uint) error
	RemoveWork(taskID, workID uint) error
	GetWorks(taskID uint, page, limit int) ([]models.Work, int64, error)
	UpdateOrders(taskIDs []uint, orderIndices []int) error
}

// taskRepository TaskRepositoryの実装
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository TaskRepositoryを作成
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create 新しいタスクを作成
func (r *taskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

// FindByID IDでタスクを検索
func (r *taskRepository) FindByID(id uint) (*models.Task, error) {
	var task models.Task
	if err := r.db.First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// Update タスク情報を更新
func (r *taskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

// Delete タスクを削除
func (r *taskRepository) Delete(id uint) error {
	return r.db.Delete(&models.Task{}, id).Error
}

// ListByProject プロジェクトのタスク一覧を取得
func (r *taskRepository) ListByProject(projectID uint) ([]models.Task, error) {
	var tasks []models.Task

	if err := r.db.Where("project_id = ?", projectID).
		Order("order_index ASC, created_at ASC").
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

// AddWork 作品をタスクに追加
func (r *taskRepository) AddWork(taskID, workID uint) error {
	taskWork := models.TaskWork{
		TaskID: taskID,
		WorkID: workID,
	}

	return r.db.Create(&taskWork).Error
}

// RemoveWork 作品をタスクから削除
func (r *taskRepository) RemoveWork(taskID, workID uint) error {
	return r.db.Where("task_id = ? AND work_id = ?", taskID, workID).Delete(&models.TaskWork{}).Error
}

// GetWorks タスクの作品一覧を取得
func (r *taskRepository) GetWorks(taskID uint, page, limit int) ([]models.Work, int64, error) {
	var works []models.Work
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&models.Work{}).
		Joins("JOIN task_works ON works.id = task_works.work_id").
		Where("task_works.task_id = ?", taskID).
		Preload("User").
		Preload("Tags")

	// 合計数を取得
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// データを取得
	if err := query.Offset(offset).Limit(limit).Order("task_works.created_at DESC").
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

// UpdateOrders タスクの表示順序を更新
func (r *taskRepository) UpdateOrders(taskIDs []uint, orderIndices []int) error {
	if len(taskIDs) != len(orderIndices) {
		return errors.New("タスクIDと順序インデックスの数が一致しません")
	}

	err := r.db.Transaction(func(tx *gorm.DB) error {
		for i, taskID := range taskIDs {
			if err := tx.Model(&models.Task{}).
				Where("id = ?", taskID).
				Update("order_index", orderIndices[i]).Error; err != nil {
				return err
			}
		}
		return nil
	})

	return err
}
