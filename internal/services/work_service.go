package services

import (
	"errors"
	"fmt"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// WorkService 作品に関するサービスインターフェース
type WorkService interface {
	Create(title, description, pdeContent, thumbnailURL string, codeShared bool, tagNames []string, taskID *uint, userID uint) (*models.Work, error)
	GetByID(id uint) (*models.Work, error)
	Update(id, userID uint, title, description, pdeContent, thumbnailURL string, codeShared bool, tagNames []string, taskID *uint) (*models.Work, error)
	Delete(id, userID uint) error
	List(page, limit int, search, tag string, userID *uint, sort string) ([]models.Work, int64, int, error)
	AddLike(userID, workID uint) (int, error)
	RemoveLike(userID, workID uint) (int, error)
	HasLiked(userID, workID uint) (bool, error)
	GetUserWorks(userID uint, page, limit int) ([]models.Work, int64, int, error)
}

// workService WorkServiceの実装
type workService struct {
	workRepo      repository.WorkRepository
	tagRepo       repository.TagRepository
	lambdaService LambdaService
	taskRepo      repository.TaskRepository
	projectRepo   repository.ProjectRepository
}

// NewWorkService WorkServiceを作成
func NewWorkService(
	workRepo repository.WorkRepository,
	tagRepo repository.TagRepository,
	lambdaService LambdaService,
	taskRepo repository.TaskRepository,
	projectRepo repository.ProjectRepository) WorkService {
	return &workService{
		workRepo:      workRepo,
		tagRepo:       tagRepo,
		lambdaService: lambdaService,
		taskRepo:      taskRepo,
		projectRepo:   projectRepo,
	}
}

// GetByID IDで作品を取得
func (s *workService) GetByID(id uint) (*models.Work, error) {
	work, err := s.workRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 閲覧数を増加
	if err := s.workRepo.IncrementViews(id); err != nil {
		// エラーでも続行
		fmt.Printf("閲覧数の更新に失敗しました: %v\n", err)
	}

	return work, nil
}

// Create 新しい作品を作成
func (s *workService) Create(
	title, description, pdeContent, thumbnailURL string,
	codeShared bool,
	tagNames []string,
	taskID *uint,
	userID uint) (*models.Work, error) {

	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// PDEコードのバリデーション
	if strings.TrimSpace(pdeContent) == "" {
		return nil, errors.New("PDEコードは必須です")
	}

	// タスクIDが指定されている場合のバリデーションと権限チェック
	if taskID != nil {
		// タスクが存在するか確認
		task, err := s.taskRepo.FindByID(*taskID)
		if err != nil {
			return nil, errors.New("指定されたタスクが見つかりません")
		}

		// ユーザーがプロジェクトのメンバーか確認
		isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
		if err != nil || !isMember {
			return nil, errors.New("このタスクに作品を投稿する権限がありません")
		}
	}

	// JavaScriptへの変換（Lambda関数を使用）
	jsContent := ""
	jsConversionErr := error(nil)

	// Lambda関数を呼び出してPDEをJSに変換
	jsContent, jsConversionErr = s.lambdaService.ConvertPDEToJS(pdeContent)
	if jsConversionErr != nil {
		// 変換に失敗しても続行するが、エラーをログ出力
		fmt.Printf("PDE変換に失敗しました: %v\n", jsConversionErr)
	}

	// 新しい作品を作成
	work := &models.Work{
		Title:             title,
		Description:       description,
		PDEContent:        pdeContent,
		JSContent:         jsContent,
		ThumbnailURL:      thumbnailURL,
		ThumbnailType:     "image/png", // TODO: URLから判定する場合は別途処理
		ThumbnailPublicID: "",          // Cloudinaryを使わない場合は不要
		CodeShared:        codeShared,
		UserID:            userID,
	}

	// データベースに保存
	if err := s.workRepo.Create(work); err != nil {
		return nil, fmt.Errorf("作品の保存に失敗しました: %v", err)
	}

	// タスクに作品を関連付け
	if taskID != nil {
		if err := s.taskRepo.AddWork(*taskID, work.ID); err != nil {
			// 作品を削除してエラーを返す
			s.workRepo.Delete(work.ID)
			return nil, fmt.Errorf("タスクへの作品の追加に失敗しました: %v", err)
		}
	}

	// タグを処理
	if len(tagNames) > 0 {
		var tagIDs []uint
		for _, name := range tagNames {
			if name = strings.TrimSpace(name); name == "" {
				continue
			}
			tag, err := s.tagRepo.FindOrCreate(name)
			if err != nil {
				continue
			}
			tagIDs = append(tagIDs, tag.ID)
		}

		if len(tagIDs) > 0 {
			if err := s.tagRepo.AttachTagsToWork(work.ID, tagIDs); err != nil {
				fmt.Printf("タグの関連付けに失敗しました: %v\n", err)
			}
		}
	}

	// JS変換に失敗した場合、非同期で再試行
	if jsConversionErr != nil {
		go func(workID uint, pdeCode string) {
			// 再度変換を試みる
			jsContent, err := s.lambdaService.ConvertPDEToJS(pdeCode)
			if err != nil {
				fmt.Printf("非同期PDE変換に失敗しました (ID=%d): %v\n", workID, err)
				return
			}

			// データベースを更新
			work, err := s.workRepo.FindByID(workID)
			if err != nil {
				fmt.Printf("作品の取得に失敗しました (ID=%d): %v\n", workID, err)
				return
			}

			work.JSContent = jsContent
			if err := s.workRepo.Update(work); err != nil {
				fmt.Printf("JS変換結果の保存に失敗しました (ID=%d): %v\n", workID, err)
			}
		}(work.ID, pdeContent)
	}

	// タグを含む作品を再取得
	return s.GetByID(work.ID)
}

// Update 作品を更新
func (s *workService) Update(id, userID uint, title, description, pdeContent, thumbnailURL string, codeShared bool, tagNames []string, taskID *uint) (*models.Work, error) {
	// 作品を取得
	work, err := s.workRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("作品が見つかりません")
	}

	// 権限チェック
	if work.UserID != userID {
		return nil, errors.New("この作品を更新する権限がありません")
	}

	// タスクIDが変更される場合の処理
	if taskID != nil {
		// 新しいタスクが存在するか確認
		task, err := s.taskRepo.FindByID(*taskID)
		if err != nil {
			return nil, errors.New("指定されたタスクが見つかりません")
		}

		// ユーザーがプロジェクトのメンバーか確認
		isMember, err := s.projectRepo.IsMember(task.ProjectID, userID)
		if err != nil || !isMember {
			return nil, errors.New("このタスクに作品を移動する権限がありません")
		}

		// 現在のタスクとの関連を削除（もしあれば）
		// 現在関連付けられているタスクを取得
		currentTasks, _, err := s.taskRepo.GetWorks(0, 1, 1) // TODO: 作品に関連するタスク一覧を取得する機能が必要
		if err == nil {
			for _, currentTask := range currentTasks {
				// 作品が含まれているタスクから削除
				s.taskRepo.RemoveWork(currentTask.ID, work.ID)
			}
		}

		// 新しいタスクに作品を追加
		if err := s.taskRepo.AddWork(*taskID, work.ID); err != nil {
			return nil, fmt.Errorf("タスクへの作品の移動に失敗しました: %v", err)
		}
	}

	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// フィールドを更新
	work.Title = title
	work.Description = description
	work.CodeShared = codeShared

	// サムネイルURLを更新
	if thumbnailURL != "" {
		work.ThumbnailURL = thumbnailURL
		work.ThumbnailType = "image/png" // TODO: URLから判定する場合は別途処理
	}

	// PDEコードが変更された場合
	pdeChanged := false
	if strings.TrimSpace(pdeContent) != "" && pdeContent != work.PDEContent {
		work.PDEContent = pdeContent
		pdeChanged = true

		// Lambda関数を呼び出してJavaScriptへの変換
		jsContent, err := s.lambdaService.ConvertPDEToJS(pdeContent)
		if err != nil {
			// 変換に失敗しても続行するが、エラーをログ出力
			fmt.Printf("PDE変換に失敗しました: %v\n", err)
		} else {
			work.JSContent = jsContent
		}
	}

	// データベースを更新
	if err := s.workRepo.Update(work); err != nil {
		return nil, fmt.Errorf("作品の更新に失敗しました: %v", err)
	}

	// タグを処理
	if tagNames != nil {
		var tagIDs []uint
		for _, name := range tagNames {
			if name = strings.TrimSpace(name); name == "" {
				continue
			}
			tag, err := s.tagRepo.FindOrCreate(name)
			if err != nil {
				continue
			}
			tagIDs = append(tagIDs, tag.ID)
		}

		// タグの関連付けを更新
		if err := s.tagRepo.AttachTagsToWork(id, tagIDs); err != nil {
			fmt.Printf("タグの更新に失敗しました: %v\n", err)
		}
	}

	// PDEが変更されていて、JS変換に失敗していれば非同期で再試行
	if pdeChanged && (work.JSContent == "" || err != nil) {
		go func(workID uint, pdeCode string) {
			// 再度変換を試みる
			jsContent, err := s.lambdaService.ConvertPDEToJS(pdeCode)
			if err != nil {
				fmt.Printf("非同期PDE変換に失敗しました (ID=%d): %v\n", workID, err)
				return
			}

			// データベースを更新
			work, err := s.workRepo.FindByID(workID)
			if err != nil {
				fmt.Printf("作品の取得に失敗しました (ID=%d): %v\n", workID, err)
				return
			}

			work.JSContent = jsContent
			if err := s.workRepo.Update(work); err != nil {
				fmt.Printf("JS変換結果の保存に失敗しました (ID=%d): %v\n", workID, err)
			}
		}(work.ID, pdeContent)
	}

	// 更新された作品を取得
	return s.GetByID(id)
}

// Delete 作品を削除
func (s *workService) Delete(id, userID uint) error {
	// 作品を取得
	work, err := s.workRepo.FindByID(id)
	if err != nil {
		return errors.New("作品が見つかりません")
	}

	// 権限チェック
	if work.UserID != userID {
		return errors.New("この作品を削除する権限がありません")
	}

	// データベースから削除
	return s.workRepo.Delete(id)
}

// List 作品一覧を取得
func (s *workService) List(page, limit int, search, tag string, userID *uint, sort string) ([]models.Work, int64, int, error) {
	works, total, err := s.workRepo.List(page, limit, search, tag, userID, sort)
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

// AddLike いいねを追加
func (s *workService) AddLike(userID, workID uint) (int, error) {
	// いいね済みかチェック
	liked, err := s.workRepo.HasLiked(userID, workID)
	if err != nil {
		return 0, err
	}

	if liked {
		return 0, errors.New("既にいいねしています")
	}

	// いいねを追加
	if err := s.workRepo.AddLike(userID, workID); err != nil {
		return 0, err
	}

	// いいね数を取得
	count, err := s.workRepo.GetLikesCount(workID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// RemoveLike いいねを削除
func (s *workService) RemoveLike(userID, workID uint) (int, error) {
	// いいね済みかチェック
	liked, err := s.workRepo.HasLiked(userID, workID)
	if err != nil {
		return 0, err
	}

	if !liked {
		return 0, errors.New("いいねしていません")
	}

	// いいねを削除
	if err := s.workRepo.RemoveLike(userID, workID); err != nil {
		return 0, err
	}

	// いいね数を取得
	count, err := s.workRepo.GetLikesCount(workID)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// HasLiked ユーザーがいいねしているか確認
func (s *workService) HasLiked(userID, workID uint) (bool, error) {
	return s.workRepo.HasLiked(userID, workID)
}

// GetUserWorks ユーザーの作品一覧を取得
func (s *workService) GetUserWorks(userID uint, page, limit int) ([]models.Work, int64, int, error) {
	works, total, err := s.workRepo.ListByUser(userID, page, limit)
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
