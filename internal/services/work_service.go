package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"
)

// WorkService 作品に関するサービスインターフェース
type WorkService interface {
	Create(title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, codeShared bool, codeContent string, tagNames []string, userID *uint, isGuest bool, guestNickname string) (*models.Work, error)
	GetByID(id uint) (*models.Work, error)
	Update(id uint, title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, codeShared bool, codeContent string, tagNames []string, userID uint) (*models.Work, error)
	Delete(id, userID uint) error
	List(page, limit int, search, tag string, userID *uint, sort string) ([]models.Work, int64, int, error)
	AddLike(userID, workID uint) (int, error)
	RemoveLike(userID, workID uint) (int, error)
	HasLiked(userID, workID uint) (bool, error)
	ListByUser(userID uint, page, limit int) ([]models.Work, int64, int, error)
	CreatePreview(file multipart.File, fileHeader *multipart.FileHeader, code string) (string, error)
}

// workService 作品サービスの実装
type workService struct {
	workRepo       repository.WorkRepository
	tagRepo        repository.TagRepository
	processingRepo repository.ProcessingRepository
	config         *config.Config
	fileService    FileService
	lambdaService  LambdaService
}

// NewWorkService WorkServiceを作成
func NewWorkService(
	workRepo repository.WorkRepository,
	tagRepo repository.TagRepository,
	processingRepo repository.ProcessingRepository,
	cfg *config.Config,
	fileService FileService,
	lambdaService LambdaService) WorkService {

	return &workService{
		workRepo:       workRepo,
		tagRepo:        tagRepo,
		processingRepo: processingRepo,
		config:         cfg,
		fileService:    fileService,
		lambdaService:  lambdaService,
	}
}

// CloudflareUploadResponse Cloudflareアップロードレスポンス
type CloudflareUploadResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	URL     string `json:"url"`
	Error   string `json:"error"`
}

// Create 新しい作品を作成
func (s *workService) Create(
	title, description string,
	file, thumbnail multipart.File,
	fileHeader, thumbnailHeader *multipart.FileHeader,
	codeShared bool,
	codeContent string,
	tagNames []string,
	userID *uint,
	isGuest bool,
	guestNickname string) (*models.Work, error) {

	// ファイルをチェック
	if file == nil {
		return nil, errors.New("ファイルが必要です")
	}

	// ファイル拡張子をチェック
	fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
	if !s.isAllowedExtension(fileExt) {
		return nil, fmt.Errorf("拡張子 %s は許可されていません", fileExt)
	}

	// ファイルサイズをチェック
	if fileHeader.Size > s.config.Storage.MaxUploadSize {
		return nil, fmt.Errorf("ファイルサイズが大きすぎます (最大 %d MB)", s.config.Storage.MaxUploadSize/1024/1024)
	}

	// ファイルURLとサムネイルURL
	var fileURL, thumbnailURL string
	var err error

	// Cloudflare Workersが有効かつ設定されている場合はCloudflareにアップロード
	if s.config.CloudflareWorker.Enabled && s.config.CloudflareWorker.URL != "" {
		fileURL, err = s.uploadToCloudflare(file, fileHeader)
		if err != nil {
			fmt.Printf("Cloudflareへのアップロードに失敗しました: %v\nローカルストレージにフォールバックします\n", err)
			// ローカルストレージにフォールバック
			fileURL, err = s.uploadToLocalStorage(file, fileHeader)
			if err != nil {
				return nil, err
			}
		}
	} else {
		// ローカルストレージにアップロード
		fileURL, err = s.uploadToLocalStorage(file, fileHeader)
		if err != nil {
			return nil, err
		}
	}

	// サムネイルをアップロード
	if thumbnail != nil && thumbnailHeader != nil {
		thumbnailURL, err = s.uploadToLocalStorage(thumbnail, thumbnailHeader)
		if err != nil {
			// サムネイルのエラーはログに残すだけで続行
			fmt.Printf("サムネイルのアップロードに失敗しました: %v\n", err)
		}
	}

	// 新しい作品を作成
	work := &models.Work{
		Title:         title,
		Description:   description,
		FileURL:       fileURL,
		ThumbnailURL:  thumbnailURL,
		CodeShared:    codeShared,
		CodeContent:   codeContent,
		UserID:        userID,
		IsGuest:       isGuest,
		GuestNickname: guestNickname,
	}

	// データベースに保存
	if err := s.workRepo.Create(work); err != nil {
		return nil, fmt.Errorf("作品の保存に失敗しました: %v", err)
	}

	// タグを処理
	if len(tagNames) > 0 {
		var tagIDs []uint
		for _, name := range tagNames {
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

	// Processing作品の場合、変換情報を登録
	if fileExt == ".pde" {
		canvasID := fmt.Sprintf("processingCanvas_%s", utils.GenerateRandomString(8))

		processingID, err := s.processingRepo.Create(&models.ProcessingWork{
			WorkID:       work.ID,
			FileName:     filepath.Base(fileURL),
			OriginalName: fileHeader.Filename,
			PDEContent:   codeContent,
			CanvasID:     canvasID,
			Status:       "pending",
		})

		if err != nil {
			fmt.Printf("Processing情報のDBへの保存に失敗しました: %v\n", err)
		} else {
			// Lambda関数を呼び出して変換処理を開始
			go func(pid uint) {
				if err := s.lambdaService.InvokePDEConversion(pid); err != nil {
					fmt.Printf("PDE変換処理に失敗しました（処理ID: %d）: %v\n", pid, err)
				}
			}(processingID)
		}
	}

	// タグを含む作品を再取得
	return s.GetByID(work.ID)
}

// Cloudflare R2にファイルをアップロード
func (s *workService) uploadToCloudflare(file multipart.File, header *multipart.FileHeader) (string, error) {
	// ファイル名を生成
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+filepath.Ext(header.Filename))

	// マルチパートリクエストを作成
	// body := &bytes.Buffer{}
	writer := http.Client{}

	// シーク位置をリセット
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("ファイルのシークに失敗しました: %v", err)
	}

	// フォームデータを作成
	formData := &bytes.Buffer{}
	formWriter := multipart.NewWriter(formData)

	// ファイルフィールドを追加
	filePart, err := formWriter.CreateFormFile("file", fileName)
	if err != nil {
		return "", fmt.Errorf("フォームファイル作成エラー: %v", err)
	}

	// ファイル内容をコピー
	if _, err = io.Copy(filePart, file); err != nil {
		return "", fmt.Errorf("ファイルコピーエラー: %v", err)
	}

	// fileName フィールドを追加
	_ = formWriter.WriteField("fileName", fileName)

	// フォームデータを閉じる
	formWriter.Close()

	// リクエストを設定
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/upload", s.config.CloudflareWorker.URL), formData)
	if err != nil {
		return "", fmt.Errorf("リクエスト作成エラー: %v", err)
	}
	req.Header.Set("Content-Type", formWriter.FormDataContentType())

	// API Keyがあれば設定
	if s.config.CloudflareWorker.APIKey != "" {
		req.Header.Set("X-API-Key", s.config.CloudflareWorker.APIKey)
	}

	// リクエストを送信
	resp, err := writer.Do(req)
	if err != nil {
		return "", fmt.Errorf("Cloudflareリクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込む
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("レスポンス読み込みエラー: %v", err)
	}

	// ステータスコードをチェック
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Cloudflareアップロード失敗: HTTP %d, レスポンス: %s", resp.StatusCode, string(respBody))
	}

	// レスポンスをパース
	var cfResp CloudflareUploadResponse
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return "", fmt.Errorf("JSONパースエラー: %v", err)
	}

	// 成功を確認
	if !cfResp.Success {
		return "", fmt.Errorf("Cloudflareアップロードエラー: %s", cfResp.Error)
	}

	// URLを返す
	return fmt.Sprintf("%s%s", s.config.CloudflareWorker.URL, cfResp.URL), nil
}

// ローカルストレージにアップロード
func (s *workService) uploadToLocalStorage(file multipart.File, header *multipart.FileHeader) (string, error) {
	// 新しいファイル名を生成
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+filepath.Ext(header.Filename))

	// ファイルサービスを使ってアップロード
	return s.fileService.UploadFile(file, fileName, "original")
}

// isAllowedExtension 許可された拡張子かチェック
func (s *workService) isAllowedExtension(ext string) bool {
	for _, allowed := range s.config.Storage.AllowedTypes {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
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
func (s *workService) Update(id uint, title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, codeShared bool, codeContent string, tagNames []string, userID uint) (*models.Work, error) {
	// 作品を取得
	work, err := s.workRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 権限チェック
	if work.UserID == nil || *work.UserID != userID {
		return nil, errors.New("この作品を更新する権限がありません")
	}

	// フィールドを更新
	work.Title = title
	work.Description = description
	work.CodeShared = codeShared
	work.CodeContent = codeContent

	// ファイルがアップロードされた場合は更新
	if file != nil && fileHeader != nil {
		// ファイル拡張子をチェック
		fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !s.isAllowedExtension(fileExt) {
			return nil, fmt.Errorf("拡張子 %s は許可されていません", fileExt)
		}

		// ファイルサイズをチェック
		if fileHeader.Size > s.config.Storage.MaxUploadSize {
			return nil, fmt.Errorf("ファイルサイズが大きすぎます (最大 %d MB)", s.config.Storage.MaxUploadSize/1024/1024)
		}

		// 新しいファイル名を生成
		fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+fileExt)

		// 新しいファイルをアップロード（Cloudflareまたはローカル）
		var fileURL string
		if s.config.CloudflareWorker.Enabled && s.config.CloudflareWorker.URL != "" {
			fileURL, err = s.uploadToCloudflare(file, fileHeader)
			if err != nil {
				// エラーの場合はローカルにフォールバック
				fileURL, err = s.uploadToLocalStorage(file, fileHeader)
				if err != nil {
					return nil, fmt.Errorf("ファイルのアップロードに失敗しました: %v", err)
				}
			}
		} else {
			fileURL, err = s.uploadToLocalStorage(file, fileHeader)
			if err != nil {
				return nil, fmt.Errorf("ファイルのアップロードに失敗しました: %v", err)
			}
		}

		// 古いファイルを削除
		if work.FileURL != "" {
			_ = s.fileService.DeleteFile(work.FileURL)
		}

		// 作品のファイルURLを更新
		work.FileURL = fileURL

		// Processing作品の場合、変換情報を更新または新規作成
		if fileExt == ".pde" {
			processingWork, err := s.processingRepo.FindByWorkID(id)
			if err != nil {
				// 既存のProcessing情報がない場合は新規作成
				canvasID := fmt.Sprintf("processingCanvas_%s", utils.GenerateRandomString(8))

				processingID, err := s.processingRepo.Create(&models.ProcessingWork{
					WorkID:       work.ID,
					FileName:     fileName,
					OriginalName: fileHeader.Filename,
					PDEContent:   codeContent,
					CanvasID:     canvasID,
					Status:       "pending",
				})

				if err != nil {
					fmt.Printf("Processing情報のDBへの保存に失敗しました: %v\n", err)
				} else {
					// バックグラウンドでJS変換を開始（非同期処理）
					go func() {
						err := s.lambdaService.InvokePDEConversion(processingID)
						if err != nil {
							fmt.Printf("PDE変換リクエストの送信に失敗しました: %v\n", err)
						}
					}()
				}
			} else {
				// 既存のProcessing情報を更新
				processingWork.FileName = fileName
				processingWork.OriginalName = fileHeader.Filename
				processingWork.PDEContent = codeContent
				processingWork.Status = "pending"
				processingWork.JSPath = ""

				if err := s.processingRepo.Update(processingWork); err != nil {
					fmt.Printf("Processing情報の更新に失敗しました: %v\n", err)
				} else {
					// Lambda関数を呼び出して変換処理を開始
					go func(pid uint) {
						if err := s.lambdaService.InvokePDEConversion(pid); err != nil {
							fmt.Printf("PDE変換処理に失敗しました（処理ID: %d）: %v\n", pid, err)
						}
					}(processingWork.ID)
				}
			}
		}
	}

	// サムネイルがアップロードされた場合は更新
	if thumbnail != nil && thumbnailHeader != nil {
		thumbnailExt := strings.ToLower(filepath.Ext(thumbnailHeader.Filename))
		thumbnailName := fmt.Sprintf("thumb_%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+thumbnailExt)

		// 新しいサムネイルをアップロード
		thumbnailURL, err := s.fileService.UploadFile(thumbnail, thumbnailName, "thumbnail")
		if err != nil {
			return nil, fmt.Errorf("サムネイルのアップロードに失敗しました: %v", err)
		}

		// 古いサムネイルを削除
		if work.ThumbnailURL != "" {
			_ = s.fileService.DeleteFile(work.ThumbnailURL)
		}

		// 作品のサムネイルURLを更新
		work.ThumbnailURL = thumbnailURL
	}

	// データベースを更新
	if err := s.workRepo.Update(work); err != nil {
		return nil, fmt.Errorf("作品の更新に失敗しました: %v", err)
	}

	// タグを処理
	if tagNames != nil {
		// 既存のタグを取得
		existingTags, err := s.tagRepo.GetTagsForWork(id)
		if err != nil {
			fmt.Printf("タグの取得に失敗しました: %v\n", err)
		} else {
			// 既存のタグIDを収集
			existingTagIDs := make(map[string]uint)
			for _, tag := range existingTags {
				existingTagIDs[tag.Name] = tag.ID
			}

			// 新しいタグ名を処理
			var newTagIDs []uint
			for _, name := range tagNames {
				if name == "" {
					continue
				}

				// 既にタグが存在するか確認
				if id, exists := existingTagIDs[name]; exists {
					delete(existingTagIDs, name) // 削除せずに残すタグをマップから削除
					newTagIDs = append(newTagIDs, id)
					continue
				}

				// 新しいタグを作成
				tag, err := s.tagRepo.FindOrCreate(name)
				if err != nil {
					fmt.Printf("タグの作成に失敗しました: %v\n", err)
					continue
				}
				newTagIDs = append(newTagIDs, tag.ID)
			}

			// マップに残っているタグは削除する
			var removeTagIDs []uint
			for _, id := range existingTagIDs {
				removeTagIDs = append(removeTagIDs, id)
			}

			// タグの関連付けを更新
			if len(removeTagIDs) > 0 {
				if err := s.tagRepo.DetachTagsFromWork(id, removeTagIDs); err != nil {
					fmt.Printf("タグの削除に失敗しました: %v\n", err)
				}
			}

			if len(newTagIDs) > 0 {
				if err := s.tagRepo.AttachTagsToWork(id, newTagIDs); err != nil {
					fmt.Printf("タグの追加に失敗しました: %v\n", err)
				}
			}
		}
	}

	// 更新された作品を取得
	return s.GetByID(id)
}

// Delete 作品を削除
func (s *workService) Delete(id, userID uint) error {
	// 作品を取得
	work, err := s.workRepo.FindByID(id)
	if err != nil {
		return err
	}

	// 権限チェック
	if work.UserID == nil || *work.UserID != userID {
		return errors.New("この作品を削除する権限がありません")
	}

	// ファイルを削除
	if work.FileURL != "" {
		_ = s.fileService.DeleteFile(work.FileURL)
	}

	// サムネイルを削除
	if work.ThumbnailURL != "" {
		_ = s.fileService.DeleteFile(work.ThumbnailURL)
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

// ListByUser ユーザーの作品一覧を取得
func (s *workService) ListByUser(userID uint, page, limit int) ([]models.Work, int64, int, error) {
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

// CreatePreview プレビューを作成
func (s *workService) CreatePreview(file multipart.File, fileHeader *multipart.FileHeader, code string) (string, error) {
	var fileName string
	if file != nil && fileHeader != nil {
		fileName = fileHeader.Filename
	} else {
		fileName = "preview.pde"
	}

	// プレビューファイルを作成
	return s.fileService.CreatePreviewFile(file, fileName, code)
}
