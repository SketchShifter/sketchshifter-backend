package repository

import (
	"errors"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"gorm.io/gorm"
)

// ProjectRepository プロジェクトに関するデータベース操作を行うインターフェース
type ProjectRepository interface {
	Create(project *models.Project) error
	FindByID(id uint) (*models.Project, error)
	FindByInvitationCode(code string) (*models.Project, error)
	Update(project *models.Project) error
	Delete(id uint) error
	List(page, limit int, search string, userID *uint) ([]models.Project, int64, error)
	AddMember(projectID, userID uint, isOwner bool) error
	RemoveMember(projectID, userID uint) error
	GetMembers(projectID uint) ([]models.ProjectMember, error)
	IsMember(projectID, userID uint) (bool, error)
	IsOwner(projectID, userID uint) (bool, error)
	GetUserProjects(userID uint, page, limit int) ([]models.Project, int64, error)
	UpdateInvitationCode(projectID uint, code string) error
}

// projectRepository ProjectRepositoryの実装
type projectRepository struct {
	db *gorm.DB
}

// NewProjectRepository ProjectRepositoryを作成
func NewProjectRepository(db *gorm.DB) ProjectRepository {
	return &projectRepository{db: db}
}

// Create 新しいプロジェクトを作成
func (r *projectRepository) Create(project *models.Project) error {
	return r.db.Create(project).Error
}

// FindByID IDでプロジェクトを検索
func (r *projectRepository) FindByID(id uint) (*models.Project, error) {
	var project models.Project
	if err := r.db.Preload("Owner").First(&project, id).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// FindByInvitationCode 招待コードでプロジェクトを検索
func (r *projectRepository) FindByInvitationCode(code string) (*models.Project, error) {
	var project models.Project
	if err := r.db.Where("invitation_code = ?", code).Preload("Owner").First(&project).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// Update プロジェクト情報を更新
func (r *projectRepository) Update(project *models.Project) error {
	return r.db.Save(project).Error
}

// Delete プロジェクトを削除
func (r *projectRepository) Delete(id uint) error {
	return r.db.Delete(&models.Project{}, id).Error
}

// List プロジェクト一覧を取得
func (r *projectRepository) List(page, limit int, search string, userID *uint) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&models.Project{}).Preload("Owner")

	// 検索条件を適用
	if search != "" {
		query = query.Where("title LIKE ? OR description LIKE ?", "%"+search+"%", "%"+search+"%")
	}

	// ユーザーIDが指定された場合は、そのユーザーが参加しているプロジェクトに限定
	if userID != nil {
		query = query.Joins("JOIN project_members ON projects.id = project_members.project_id").
			Where("project_members.user_id = ?", *userID)
	}

	// 合計数を取得
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// データを取得
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&projects).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, 0, err
		}
	}

	return projects, total, nil
}

// AddMember メンバーをプロジェクトに追加
func (r *projectRepository) AddMember(projectID, userID uint, isOwner bool) error {
	member := models.ProjectMember{
		ProjectID: projectID,
		UserID:    userID,
		IsOwner:   isOwner,
	}

	return r.db.Create(&member).Error
}

// RemoveMember メンバーをプロジェクトから削除
func (r *projectRepository) RemoveMember(projectID, userID uint) error {
	return r.db.Where("project_id = ? AND user_id = ?", projectID, userID).Delete(&models.ProjectMember{}).Error
}

// GetMembers プロジェクトのメンバー一覧を取得
func (r *projectRepository) GetMembers(projectID uint) ([]models.ProjectMember, error) {
	var members []models.ProjectMember

	if err := r.db.Where("project_id = ?", projectID).
		Preload("User").
		Find(&members).Error; err != nil {
		return nil, err
	}

	return members, nil
}

// IsMember ユーザーがプロジェクトのメンバーかどうか確認
func (r *projectRepository) IsMember(projectID, userID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// IsOwner ユーザーがプロジェクトのオーナーかどうか確認
func (r *projectRepository) IsOwner(projectID, userID uint) (bool, error) {
	var count int64
	if err := r.db.Model(&models.ProjectMember{}).
		Where("project_id = ? AND user_id = ? AND is_owner = true", projectID, userID).
		Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

// GetUserProjects ユーザーが参加しているプロジェクト一覧を取得
func (r *projectRepository) GetUserProjects(userID uint, page, limit int) ([]models.Project, int64, error) {
	var projects []models.Project
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&models.Project{}).
		Joins("JOIN project_members ON projects.id = project_members.project_id").
		Where("project_members.user_id = ?", userID).
		Preload("Owner")

	// 合計数を取得
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// データを取得
	if err := query.Offset(offset).Limit(limit).Order("projects.created_at DESC").
		Find(&projects).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, 0, err
	}

	return projects, total, nil
}

// UpdateInvitationCode 招待コードを更新
func (r *projectRepository) UpdateInvitationCode(projectID uint, code string) error {
	return r.db.Model(&models.Project{}).
		Where("id = ?", projectID).
		Update("invitation_code", code).Error
}
