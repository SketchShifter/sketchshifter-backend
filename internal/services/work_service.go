package services

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
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

// workService WorkServiceの実装
type workService struct {
	workRepo       repository.WorkRepository
	tagRepo        repository.TagRepository
	imageRepo      repository.ImageRepository
	processingRepo repository.ProcessingRepository
	config         *config.Config
	fileUtils      utils.FileUtils
	awsSession     *session.Session
}

// NewWorkService WorkServiceを作成
func NewWorkService(
	workRepo repository.WorkRepository,
	tagRepo repository.TagRepository,
	imageRepo repository.ImageRepository,
	processingRepo repository.ProcessingRepository,
	cfg *config.Config,
	fileUtils utils.FileUtils) WorkService {

	// AWSセッションの初期化
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWS.Region),
	})

	if err != nil {
		// エラーハンドリング（本番環境では適切なエラーハンドリングを行うこと）
		fmt.Printf("AWS Session initialization error: %v\n", err)
	}

	return &workService{
		workRepo:       workRepo,
		tagRepo:        tagRepo,
		imageRepo:      imageRepo,
		processingRepo: processingRepo,
		config:         cfg,
		fileUtils:      fileUtils,
		awsSession:     awsSession,
	}
}

// Create 新しい作品を作成
func (s *workService) Create(title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, codeShared bool, codeContent string, tagNames []string, userID *uint, isGuest bool, guestNickname string) (*models.Work, error) {
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

	// 新しいファイル名を生成
	fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+fileExt)

	// CloudflareワーカーにファイルをアップロードするためのURLを取得
	fileURL, err := s.uploadToCloudflareR2(file, fileName, "original")
	if err != nil {
		return nil, fmt.Errorf("ファイルのアップロードに失敗しました: %v", err)
	}

	// サムネイルをアップロード
	var thumbnailURL string
	if thumbnail != nil && thumbnailHeader != nil {
		thumbnailExt := strings.ToLower(filepath.Ext(thumbnailHeader.Filename))
		thumbnailName := fmt.Sprintf("thumb_%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+thumbnailExt)

		thumbnailURL, err = s.uploadToCloudflareR2(thumbnail, thumbnailName, "original")
		if err != nil {
			return nil, fmt.Errorf("サムネイルのアップロードに失敗しました: %v", err)
		}

		// サムネイルの画像変換キューに追加
		if err := s.queueImageForConversion(thumbnailName, "thumbnail"); err != nil {
			// エラーがあっても処理は続行
			fmt.Printf("サムネイル変換キューの登録に失敗しました: %v\n", err)
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
		return nil, err
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
				return nil, err
			}
		}
	}

	// Processing作品の場合、変換キューに追加
	if fileExt == ".pde" {
		if err := s.processPDEFile(work.ID, fileName, fileHeader.Filename, fileURL); err != nil {
			// エラーがあっても作品の作成自体は成功とする
			fmt.Printf("PDE変換キューの登録に失敗しました: %v\n", err)
		}
	} else if isImageFile(fileExt) {
		// 画像ファイルの場合、変換キューに追加
		if err := s.queueImageForConversion(fileName, "work"); err != nil {
			// エラーがあっても処理は続行
			fmt.Printf("画像変換キューの登録に失敗しました: %v\n", err)
		}
	}

	// タグを含む作品を再取得
	return s.GetByID(work.ID)
}

// uploadToCloudflareR2 ファイルをCloudflare R2にアップロード
func (s *workService) uploadToCloudflareR2(file multipart.File, fileName, fileType string) (string, error) {
	// ファイルを最初の位置に戻す
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// ファイル内容を読み込む
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// マルチパートフォームデータを作成
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// ファイルフィールドを追加
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", err
	}

	if _, err := part.Write(fileBytes); err != nil {
		return "", err
	}

	// タイプフィールドを追加
	if err := writer.WriteField("type", fileType); err != nil {
		return "", err
	}

	// ファイル名フィールドを追加
	if err := writer.WriteField("fileName", fileName); err != nil {
		return "", err
	}

	writer.Close()

	// Cloudflare Workerにリクエスト
	req, err := http.NewRequest("POST", s.config.Cloudflare.WorkerURL+"/upload", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", s.config.Cloudflare.APIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// エラーレスポンスの内容を読み取り
		errorBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Cloudflare Workerが失敗しました: %s - %s", resp.Status, string(errorBody))
	}

	// レスポンスをパース
	var result struct {
		Success bool   `json:"success"`
		Path    string `json:"path"`
		URL     string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if !result.Success {
		return "", errors.New("Cloudflare Workerのレスポンスがエラーを示しています")
	}

	// URLを返す
	return s.config.Cloudflare.WorkerURL + result.URL, nil
}

// processPDEFile メソッドの変更
func (s *workService) processPDEFile(workID uint, fileName, originalName string, pdeContent string) error {
	// SQSクライアントを初期化
	sqsSvc := sqs.New(s.awsSession)

	// キューURLを設定
	queueURL := s.config.AWS.PdeConversionQueueURL

	// DBに処理情報を保存
	canvasID := "processingCanvas_" + utils.GenerateRandomString(8)

	processingID, err := s.processingRepo.Create(&models.ProcessingWork{
		WorkID:       workID,
		FileName:     fileName,
		OriginalName: originalName,
		PDEContent:   pdeContent, // PDEファイルの内容を直接保存
		CanvasID:     canvasID,
		Status:       "pending",
	})

	if err != nil {
		return fmt.Errorf("Processing情報のDBへの保存に失敗しました: %v", err)
	}

	// メッセージ内容を作成
	messageBody := struct {
		Type           string `json:"type"`
		ProcessingData struct {
			ID           uint   `json:"id"`
			WorkID       uint   `json:"workID"`
			FileName     string `json:"fileName"`
			OriginalName string `json:"originalName"`
			PdeContent   string `json:"pdeContent"` // PDEファイルの内容を直接渡す
			CanvasID     string `json:"canvasId"`
		} `json:"processingData"`
	}{
		Type: "pde_conversion",
	}

	messageBody.ProcessingData.ID = processingID
	messageBody.ProcessingData.WorkID = workID
	messageBody.ProcessingData.FileName = fileName
	messageBody.ProcessingData.OriginalName = originalName
	messageBody.ProcessingData.PdeContent = pdeContent
	messageBody.ProcessingData.CanvasID = canvasID

	messageJSON, err := json.Marshal(messageBody)
	if err != nil {
		return err
	}

	// SQSにメッセージを送信
	_, err = sqsSvc.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(messageJSON)),
	})

	return err
}

// queueImageForConversion メソッドの変更
func (s *workService) queueImageForConversion(fileName, imageType string) error {
	// SQSクライアントを初期化
	sqsSvc := sqs.New(s.awsSession)

	// キューURLを設定
	queueURL := s.config.AWS.WebpConversionQueueURL

	// DBに画像情報を保存
	imageID, err := s.imageRepo.Create(&models.Image{
		FileName:     fileName,
		OriginalPath: "original/" + fileName,
		Status:       "pending",
	})

	if err != nil {
		return fmt.Errorf("画像情報のDBへの保存に失敗しました: %v", err)
	}

	// メッセージ内容を作成
	messageBody := struct {
		Type      string `json:"type"`
		ImageData struct {
			ID        uint   `json:"id"`
			FileName  string `json:"fileName"`
			ImageType string `json:"imageType"`
		} `json:"imageData"`
	}{
		Type: "webp_conversion",
	}

	messageBody.ImageData.ID = imageID
	messageBody.ImageData.FileName = fileName
	messageBody.ImageData.ImageType = imageType

	messageJSON, err := json.Marshal(messageBody)
	if err != nil {
		return err
	}

	// SQSにメッセージを送信
	_, err = sqsSvc.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(queueURL),
		MessageBody: aws.String(string(messageJSON)),
	})

	return err
}

// isImageFile 画像ファイルかどうかをチェック
func isImageFile(ext string) bool {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	for _, imageExt := range imageExts {
		if ext == imageExt {
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
	}

	return work, nil
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

// 他のメソッドは変更なしで維持
// Update, Delete, List, AddLike, RemoveLike, HasLiked, ListByUser, CreatePreview

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

		// 新しいファイルを保存
		fileName := fmt.Sprintf("%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+fileExt)
		filePath := filepath.Join(s.config.Storage.UploadDir, fileName)
		fileURL, err := s.fileUtils.SaveFile(file, filePath)
		if err != nil {
			return nil, err
		}

		// 古いファイルを削除
		if work.FileURL != "" {
			_ = os.Remove(filepath.Join(s.config.Storage.UploadDir, filepath.Base(work.FileURL)))
		}

		work.FileURL = fileURL
	}

	// サムネイルがアップロードされた場合は更新
	if thumbnail != nil && thumbnailHeader != nil {
		thumbnailExt := strings.ToLower(filepath.Ext(thumbnailHeader.Filename))
		thumbnailName := fmt.Sprintf("thumb_%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+thumbnailExt)
		thumbnailPath := filepath.Join(s.config.Storage.UploadDir, thumbnailName)
		thumbnailURL, err := s.fileUtils.SaveFile(thumbnail, thumbnailPath)
		if err != nil {
			return nil, err
		}

		// 古いサムネイルを削除
		if work.ThumbnailURL != "" {
			_ = os.Remove(filepath.Join(s.config.Storage.UploadDir, filepath.Base(work.ThumbnailURL)))
		}

		work.ThumbnailURL = thumbnailURL
	}

	// データベースを更新
	if err := s.workRepo.Update(work); err != nil {
		return nil, err
	}

	// タグを処理
	if tagNames != nil {
		// 既存のタグを取得
		existingTags, err := s.tagRepo.GetTagsForWork(id)
		if err != nil {
			return nil, err
		}

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
				return nil, err
			}
		}

		if len(newTagIDs) > 0 {
			if err := s.tagRepo.AttachTagsToWork(id, newTagIDs); err != nil {
				return nil, err
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
		_ = os.Remove(filepath.Join(s.config.Storage.UploadDir, filepath.Base(work.FileURL)))
	}

	// サムネイルを削除
	if work.ThumbnailURL != "" {
		_ = os.Remove(filepath.Join(s.config.Storage.UploadDir, filepath.Base(work.ThumbnailURL)))
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
	// ファイルをチェック
	if file == nil && code == "" {
		return "", errors.New("ファイルまたはコードが必要です")
	}

	var previewPath string
	var previewURL string

	// ファイルがある場合はアップロード
	if file != nil && fileHeader != nil {
		// ファイル拡張子をチェック
		fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !s.isAllowedExtension(fileExt) {
			return "", fmt.Errorf("拡張子 %s は許可されていません", fileExt)
		}

		// 一時ファイルとして保存
		fileName := fmt.Sprintf("preview_%d_%s", time.Now().Unix(), utils.GenerateRandomString(8)+fileExt)
		previewPath = filepath.Join(s.config.Storage.UploadDir, "preview", fileName)

		// ディレクトリが存在することを確認
		if err := os.MkdirAll(filepath.Join(s.config.Storage.UploadDir, "preview"), 0755); err != nil {
			return "", err
		}

		var err error
		previewURL, err = s.fileUtils.SaveFile(file, previewPath)
		if err != nil {
			return "", err
		}
	} else if code != "" {
		// コードからファイルを作成
		fileName := fmt.Sprintf("preview_%d_%s.pde", time.Now().Unix(), utils.GenerateRandomString(8))
		previewPath = filepath.Join(s.config.Storage.UploadDir, "preview", fileName)

		// ディレクトリが存在することを確認
		if err := os.MkdirAll(filepath.Join(s.config.Storage.UploadDir, "preview"), 0755); err != nil {
			return "", err
		}

		// コードをファイルに書き込み
		if err := os.WriteFile(previewPath, []byte(code), 0644); err != nil {
			return "", err
		}

		// URLを設定
		previewURL = "/uploads/preview/" + fileName
	}

	// 一定時間後にプレビューファイルを削除するゴルーチンを起動
	go func() {
		time.Sleep(1 * time.Hour)
		_ = os.Remove(previewPath)
	}()

	return previewURL, nil
}
