package services

import (
	"errors"
	"fmt"
	"mime/multipart"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"
)

// WorkService 作品に関するサービスインターフェース
type WorkService interface {
	Create(title, description string, pdeContent string, thumbnail multipart.File, thumbnailHeader *multipart.FileHeader, codeShared bool, tagNames []string, userID uint) (*models.Work, error)
	GetByID(id uint) (*models.Work, error)
	Update(id, userID uint, title, description string, pdeContent string, thumbnail multipart.File, thumbnailHeader *multipart.FileHeader, codeShared bool, tagNames []string) (*models.Work, error)
	Delete(id, userID uint) error
	List(page, limit int, search, tag string, userID *uint, sort string) ([]models.Work, int64, int, error)
	AddLike(userID, workID uint) (int, error)
	RemoveLike(userID, workID uint) (int, error)
	HasLiked(userID, workID uint) (bool, error)
	GetUserWorks(userID uint, page, limit int) ([]models.Work, int64, int, error)
}

// workService WorkServiceの実装
type workService struct {
	workRepo          repository.WorkRepository
	tagRepo           repository.TagRepository
	cloudinaryService CloudinaryService
	lambdaService     LambdaService
}

// NewWorkService WorkServiceを作成
func NewWorkService(
	workRepo repository.WorkRepository,
	tagRepo repository.TagRepository,
	cloudinaryService CloudinaryService,
	lambdaService LambdaService) WorkService {
	return &workService{
		workRepo:          workRepo,
		tagRepo:           tagRepo,
		cloudinaryService: cloudinaryService,
		lambdaService:     lambdaService,
	}
}

// Create 新しい作品を作成
func (s *workService) Create(
	title, description string,
	pdeContent string,
	thumbnail multipart.File,
	thumbnailHeader *multipart.FileHeader,
	codeShared bool,
	tagNames []string,
	userID uint) (*models.Work, error) {

	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// PDEコードのバリデーション
	if strings.TrimSpace(pdeContent) == "" {
		return nil, errors.New("PDEコードは必須です")
	}

	// 変数の準備
	var thumbnailURL, thumbnailPublicID, thumbnailType string
	var err error

	// サムネイルがある場合は処理
	if thumbnail != nil && thumbnailHeader != nil {
		// サムネイルは画像データを確認
		if !isImageFile(thumbnailHeader.Filename) {
			return nil, errors.New("サムネイルは画像ファイル（PNG, JPEG, GIF, WebP）である必要があります")
		}

		// Cloudinaryにアップロード
		thumbnailPublicID, thumbnailURL, err = s.cloudinaryService.UploadImage(
			thumbnail,
			utils.GenerateRandomString(8)+"_thumb_"+thumbnailHeader.Filename,
			70)
		if err != nil {
			return nil, fmt.Errorf("サムネイルのアップロードに失敗しました: %v", err)
		}

		thumbnailType = getContentTypeFromFilename(thumbnailHeader.Filename)
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
		ThumbnailType:     thumbnailType,
		ThumbnailPublicID: thumbnailPublicID,
		CodeShared:        codeShared,
		UserID:            userID,
	}

	// データベースに保存
	if err := s.workRepo.Create(work); err != nil {
		// エラーが発生した場合、Cloudinaryにアップロードした画像を削除
		if thumbnailPublicID != "" {
			s.cloudinaryService.DeleteImage(thumbnailPublicID)
		}
		return nil, fmt.Errorf("作品の保存に失敗しました: %v", err)
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

// Update 作品を更新
func (s *workService) Update(id, userID uint, title, description string, pdeContent string, thumbnail multipart.File, thumbnailHeader *multipart.FileHeader, codeShared bool, tagNames []string) (*models.Work, error) {
	// 作品を取得
	work, err := s.workRepo.FindByID(id)
	if err != nil {
		return nil, errors.New("作品が見つかりません")
	}

	// 権限チェック
	if work.UserID != userID {
		return nil, errors.New("この作品を更新する権限がありません")
	}

	// タイトルのバリデーション
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("タイトルは必須です")
	}

	// フィールドを更新
	work.Title = title
	work.Description = description
	work.CodeShared = codeShared

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

	// サムネイルがアップロードされた場合は更新
	if thumbnail != nil && thumbnailHeader != nil {
		// サムネイルは画像データを確認
		if !isImageFile(thumbnailHeader.Filename) {
			return nil, errors.New("サムネイルは画像ファイル（PNG, JPEG, GIF, WebP）である必要があります")
		}

		// 古いサムネイルがあれば削除予定としてマーク
		oldThumbnailPublicID := work.ThumbnailPublicID

		// Cloudinaryにアップロード
		thumbnailPublicID, thumbnailURL, err := s.cloudinaryService.UploadImage(
			thumbnail,
			utils.GenerateRandomString(8)+"_thumb_"+thumbnailHeader.Filename,
			70)
		if err != nil {
			return nil, fmt.Errorf("サムネイルのアップロードに失敗しました: %v", err)
		}

		// 新しいURLを設定
		work.ThumbnailURL = thumbnailURL
		work.ThumbnailPublicID = thumbnailPublicID
		work.ThumbnailType = getContentTypeFromFilename(thumbnailHeader.Filename)

		// 古いサムネイルがあれば削除
		if oldThumbnailPublicID != "" {
			s.cloudinaryService.DeleteImage(oldThumbnailPublicID)
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

	// Cloudinaryから画像を削除
	if work.ThumbnailPublicID != "" {
		if err := s.cloudinaryService.DeleteImage(work.ThumbnailPublicID); err != nil {
			fmt.Printf("サムネイルの削除に失敗しました: %v\n", err)
		}
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

// isImageFile ファイル名が画像ファイルかどうかを判定
func isImageFile(filename string) bool {
	ext := strings.ToLower(getFileExtension(filename))
	for _, imgExt := range []string{".png", ".jpg", ".jpeg", ".gif", ".webp"} {
		if ext == imgExt {
			return true
		}
	}
	return false
}

// getFileExtension ファイル名から拡張子を取得
func getFileExtension(filename string) string {
	parts := strings.Split(filename, ".")
	if len(parts) < 2 {
		return ""
	}
	return "." + parts[len(parts)-1]
}

// getContentTypeFromFilename ファイル名からContent-Typeを取得
func getContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(getFileExtension(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pde":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}
