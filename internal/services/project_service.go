package services

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// ProjectService プロジェクトに関するサービスインターフェース
type ProjectService interface {
	Create(title, description string, userID uint) (*models.Project, error)
	GetByID(id uint) (*models.Project, error)
	Update(id, userID uint, title, description string) (*models.Project, error)
	Delete(id, userID uint) error
	List(page, limit int, search string, userID *uint) ([]models.Project, int64, int, error)
	GetMembers(projectID uint) ([]models.ProjectMember, error)
	AddMember(projectID, userID uint, isOwner bool) error
	RemoveMember(projectID, ownerID, userID uint) error
	JoinByInvitationCode(code string, userID uint) (*models.Project, error)
	GenerateInvitationCode(projectID, userID uint) (string, error)
	IsUserAllowed(projectID, userID uint) (bool, error)
	IsOwner(projectID, userID uint) (bool, error)
	GetUserProjects(userID uint, page, limit int) ([]models.Project, int64, int, error)
}

// projectService ProjectServiceの実装
type projectService struct {
	projectRepo repository.ProjectRepository
	taskRepo    repository.TaskRepository
}

// NewProjectService ProjectServiceを作成
func NewProjectService(projectRepo repository.ProjectRepository, taskRepo repository.TaskRepository) ProjectService {
	return &projectService{
		projectRepo: projectRepo,
		taskRepo:    taskRepo,
	}
}

// Create 新しいプロジェクトを作成
func (s *projectService) Create(title, description string, userID uint) (*models.Project, error) {
	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// 招待コードを生成
	code := generateInvitationCode()

	// プロジェクトを作成
	project := &models.Project{
		Title:          title,
		Description:    description,
		OwnerID:        userID,
		InvitationCode: code,
	}

	// データベースに保存
	if err := s.projectRepo.Create(project); err != nil {
		return nil, fmt.Errorf("プロジェクトの作成に失敗しました: %v", err)
	}

	// 作成者をオーナーとしてメンバーに追加
	if err := s.projectRepo.AddMember(project.ID, userID, true); err != nil {
		return nil, fmt.Errorf("オーナー情報の登録に失敗しました: %v", err)
	}

	return s.GetByID(project.ID)
}

// GetByID IDでプロジェクトを取得
func (s *projectService) GetByID(id uint) (*models.Project, error) {
	project, err := s.projectRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("プロジェクトが見つかりません")
	}

	// タスク一覧を取得してセット
	tasks, err := s.taskRepo.ListByProject(id)
	if err == nil {
		project.Tasks = tasks
	}

	return project, nil
}

// Update プロジェクトを更新
func (s *projectService) Update(id, userID uint, title, description string) (*models.Project, error) {
	// プロジェクトを取得
	project, err := s.projectRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("プロジェクトが見つかりません")
	}

	// 権限チェック
	isOwner, err := s.projectRepo.IsOwner(id, userID)
	if err != nil || !isOwner {
		return nil, errors.New("このプロジェクトを更新する権限がありません")
	}

	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// フィールドを更新
	project.Title = title
	project.Description = description

	// データベースを更新
	if err := s.projectRepo.Update(project); err != nil {
		return nil, fmt.Errorf("プロジェクトの更新に失敗しました: %v", err)
	}

	return s.GetByID(id)
}

// Delete プロジェクトを削除
func (s *projectService) Delete(id, userID uint) error {
	// プロジェクトを取得
	_, err := s.projectRepo.FindByID(id)
	if err != nil {
		return errors.New("プロジェクトが見つかりません")
	}

	// 権限チェック
	isOwner, err := s.projectRepo.IsOwner(id, userID)
	if err != nil || !isOwner {
		return errors.New("このプロジェクトを削除する権限がありません")
	}

	// プロジェクトを削除
	if err := s.projectRepo.Delete(id); err != nil {
		return fmt.Errorf("プロジェクトの削除に失敗しました: %v", err)
	}

	return nil
}

// List プロジェクト一覧を取得
func (s *projectService) List(page, limit int, search string, userID *uint) ([]models.Project, int64, int, error) {
	projects, total, err := s.projectRepo.List(page, limit, search, userID)
	if err != nil {
		return nil, 0, 0, err
	}

	// 総ページ数を計算
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}

	return projects, total, pages, nil
}

// GetMembers プロジェクトのメンバー一覧を取得
func (s *projectService) GetMembers(projectID uint) ([]models.ProjectMember, error) {
	// プロジェクトが存在するか確認
	_, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return nil, errors.New("プロジェクトが見つかりません")
	}

	// メンバー一覧を取得
	return s.projectRepo.GetMembers(projectID)
}

// AddMember メンバーをプロジェクトに追加
func (s *projectService) AddMember(projectID, userID uint, isOwner bool) error {
	// プロジェクトが存在するか確認
	_, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return errors.New("プロジェクトが見つかりません")
	}

	// 既にメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(projectID, userID)
	if err != nil {
		return err
	}

	if isMember {
		return errors.New("このユーザーは既にメンバーです")
	}

	// メンバーを追加
	return s.projectRepo.AddMember(projectID, userID, isOwner)
}

// RemoveMember メンバーをプロジェクトから削除
func (s *projectService) RemoveMember(projectID, ownerID, userID uint) error {
	// プロジェクトが存在するか確認
	_, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return errors.New("プロジェクトが見つかりません")
	}

	// 権限チェック
	isOwner, err := s.projectRepo.IsOwner(projectID, ownerID)
	if err != nil || !isOwner {
		return errors.New("このプロジェクトからメンバーを削除する権限がありません")
	}

	// オーナーが自分自身を削除しようとしていないか確認
	if ownerID == userID {
		return errors.New("オーナーは自分自身をプロジェクトから削除できません")
	}

	// メンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(projectID, userID)
	if err != nil {
		return err
	}

	if !isMember {
		return errors.New("このユーザーはメンバーではありません")
	}

	// メンバーを削除
	return s.projectRepo.RemoveMember(projectID, userID)
}

// JoinByInvitationCode 招待コードを使用してプロジェクトに参加
func (s *projectService) JoinByInvitationCode(code string, userID uint) (*models.Project, error) {
	// 招待コードが有効かどうか確認
	project, err := s.projectRepo.FindByInvitationCode(code)
	if err != nil {
		return nil, errors.New("無効な招待コードです")
	}

	// 既にメンバーかどうか確認
	isMember, err := s.projectRepo.IsMember(project.ID, userID)
	if err != nil {
		return nil, err
	}

	if isMember {
		return nil, errors.New("あなたは既にこのプロジェクトのメンバーです")
	}

	// メンバーとして追加（オーナーではない）
	if err := s.projectRepo.AddMember(project.ID, userID, false); err != nil {
		return nil, fmt.Errorf("プロジェクトへの参加に失敗しました: %v", err)
	}

	return s.GetByID(project.ID)
}

// GenerateInvitationCode 新しい招待コードを生成
func (s *projectService) GenerateInvitationCode(projectID, userID uint) (string, error) {
	// プロジェクトが存在するか確認
	_, err := s.projectRepo.FindByID(projectID)
	if err != nil {
		return "", errors.New("プロジェクトが見つかりません")
	}

	// 権限チェック
	isOwner, err := s.projectRepo.IsOwner(projectID, userID)
	if err != nil || !isOwner {
		return "", errors.New("招待コードを生成する権限がありません")
	}

	// 新しい招待コードを生成
	code := generateInvitationCode()

	// 招待コードを更新
	if err := s.projectRepo.UpdateInvitationCode(projectID, code); err != nil {
		return "", fmt.Errorf("招待コードの更新に失敗しました: %v", err)
	}

	return code, nil
}

// IsUserAllowed ユーザーがプロジェクトにアクセスできるか確認
func (s *projectService) IsUserAllowed(projectID, userID uint) (bool, error) {
	return s.projectRepo.IsMember(projectID, userID)
}

// IsOwner ユーザーがプロジェクトのオーナーかどうか確認
func (s *projectService) IsOwner(projectID, userID uint) (bool, error) {
	return s.projectRepo.IsOwner(projectID, userID)
}

// GetUserProjects ユーザーが参加しているプロジェクト一覧を取得
func (s *projectService) GetUserProjects(userID uint, page, limit int) ([]models.Project, int64, int, error) {
	projects, total, err := s.projectRepo.GetUserProjects(userID, page, limit)
	if err != nil {
		return nil, 0, 0, err
	}

	// 総ページ数を計算
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}

	return projects, total, pages, nil
}

// generateInvitationCode ランダムな招待コードを生成する
func generateInvitationCode() string {
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789" // 似た文字（0/O, 1/I）を除外
	const length = 8

	// 乱数生成器の初期化
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	code := make([]byte, length)
	for i := range code {
		code[i] = charset[r.Intn(len(charset))]
	}

	return string(code)
}
