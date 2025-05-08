package services

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"
)

// WorkService 作品に関するサービスインターフェース
type WorkService interface {
	Create(title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, codeShared bool, codeContent string, tagNames []string, userID *uint, isGuest bool, guestNickname string) (*models.Work, error)
	CreateWithCloudinary(title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, fileURL, filePublicID, fileType, fileName, thumbnailURL, thumbnailPublicID, thumbnailType string, codeShared bool, codeContent string, tagNames []string, userID *uint, isGuest bool, guestNickname string) (*models.Work, error)
	GetByID(id uint) (*models.Work, error)
	Update(id uint, title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, codeShared bool, codeContent string, tagNames []string, userID uint) (*models.Work, error)
	UpdateWithCloudinary(id uint, title, description string, file, thumbnail multipart.File, fileHeader, thumbnailHeader *multipart.FileHeader, fileURL, filePublicID, fileType, fileName, thumbnailURL, thumbnailPublicID, thumbnailType string, codeShared bool, codeContent string, tagNames []string, userID uint) (*models.Work, error)
	Delete(id, userID uint) error
	List(page, limit int, search, tag string, userID *uint, sort string) ([]models.Work, int64, int, error)
	AddLike(userID, workID uint) (int, error)
	RemoveLike(userID, workID uint) (int, error)
	HasLiked(userID, workID uint) (bool, error)
	GetFileData(id uint) ([]byte, string, string, error)
	GetThumbnailData(id uint) ([]byte, string, error)
	CreatePreview(file multipart.File, fileHeader *multipart.FileHeader, code string) ([]byte, string, error)
	GetProcessingWorkByWorkID(workID uint) (*models.ProcessingWork, error)
}

// workService WorkServiceの実装
type workService struct {
	workRepo          repository.WorkRepository
	tagRepo           repository.TagRepository
	processingRepo    repository.ProcessingRepository
	lambdaService     LambdaService
	cloudinaryService CloudinaryService
}

// NewWorkService WorkServiceを作成
func NewWorkService(
	workRepo repository.WorkRepository,
	tagRepo repository.TagRepository,
	processingRepo repository.ProcessingRepository,
	lambdaService LambdaService,
	cloudinaryService CloudinaryService) WorkService {
	return &workService{
		workRepo:          workRepo,
		tagRepo:           tagRepo,
		processingRepo:    processingRepo,
		lambdaService:     lambdaService,
		cloudinaryService: cloudinaryService,
	}
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

	// ファイルタイプを決定
	fileType := s.getContentType(fileExt)

	// 変数の準備
	var fileData []byte
	var thumbnailData []byte
	var thumbnailType string
	var fileURL, filePublicID string
	var thumbnailURL, thumbnailPublicID string
	var err error

	// 画像ファイルの場合は Cloudinary にアップロード
	if s.isImageExtension(fileExt) {
		// ファイルポインタを先頭に戻す
		if _, err := file.Seek(0, 0); err != nil {
			return nil, fmt.Errorf("ファイルの読み込み準備に失敗しました: %v", err)
		}

		// Cloudinaryにアップロード (品質75%で圧縮)
		filePublicID, fileURL, err = s.cloudinaryService.UploadImage(file, utils.GenerateRandomString(8)+"_"+fileHeader.Filename, 75)
		if err != nil {
			return nil, fmt.Errorf("画像のアップロードに失敗しました: %v", err)
		}
	} else {
		// 画像以外のファイルはDB に保存
		fileData, err = io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("ファイルの読み込みに失敗しました: %v", err)
		}
	}

	// サムネイルがある場合は処理
	if thumbnail != nil && thumbnailHeader != nil {
		thumbnailExt := strings.ToLower(filepath.Ext(thumbnailHeader.Filename))
		thumbnailType = s.getContentType(thumbnailExt)

		// サムネイルは常に画像なので Cloudinary にアップロード
		if _, err := thumbnail.Seek(0, 0); err != nil {
			// Cloudinary にアップロード済みのファイルがあれば削除
			if filePublicID != "" {
				_ = s.cloudinaryService.DeleteImage(filePublicID)
			}
			return nil, fmt.Errorf("サムネイルの読み込み準備に失敗しました: %v", err)
		}

		// Cloudinaryにアップロード (品質70%で圧縮)
		thumbnailPublicID, thumbnailURL, err = s.cloudinaryService.UploadImage(
			thumbnail,
			utils.GenerateRandomString(8)+"_thumb_"+thumbnailHeader.Filename,
			70)
		if err != nil {
			// Cloudinary にアップロード済みのファイルがあれば削除
			if filePublicID != "" {
				_ = s.cloudinaryService.DeleteImage(filePublicID)
			}
			return nil, fmt.Errorf("サムネイルのアップロードに失敗しました: %v", err)
		}
	}

	// 新しい作品を作成
	work := &models.Work{
		Title:             title,
		Description:       description,
		FileData:          fileData,
		FileType:          fileType,
		FileName:          fileHeader.Filename,
		FileURL:           fileURL,
		FilePublicID:      filePublicID,
		ThumbnailData:     thumbnailData,
		ThumbnailType:     thumbnailType,
		ThumbnailURL:      thumbnailURL,
		ThumbnailPublicID: thumbnailPublicID,
		CodeShared:        codeShared,
		CodeContent:       codeContent,
		UserID:            userID,
		IsGuest:           isGuest,
		GuestNickname:     guestNickname,
	}

	// データベースに保存
	if err := s.workRepo.Create(work); err != nil {
		// エラーが発生した場合、Cloudinary にアップロードした画像を削除
		if filePublicID != "" {
			_ = s.cloudinaryService.DeleteImage(filePublicID)
		}
		if thumbnailPublicID != "" {
			_ = s.cloudinaryService.DeleteImage(thumbnailPublicID)
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

	// Processing作品の場合、変換情報を登録
	if fileExt == ".pde" {
		canvasID := fmt.Sprintf("processingCanvas_%s", utils.GenerateRandomString(8))

		// PDEコードがなければファイルから読み込む
		if codeContent == "" {
			codeContent = string(fileData)
		}

		processingID, err := s.processingRepo.Create(&models.ProcessingWork{
			WorkID:       work.ID,
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

// CreateWithCloudinary フロントエンドから直接Cloudinaryにアップロードされたファイル情報を使用して作品を作成
func (s *workService) CreateWithCloudinary(
	title, description string,
	file, thumbnail multipart.File,
	fileHeader, thumbnailHeader *multipart.FileHeader,
	fileURL, filePublicID, fileType, fileName,
	thumbnailURL, thumbnailPublicID, thumbnailType string,
	codeShared bool,
	codeContent string,
	tagNames []string,
	userID *uint,
	isGuest bool,
	guestNickname string) (*models.Work, error) {

	var fileData []byte
	var err error

	// PDEファイルがアップロードされた場合（Cloudinaryを使用しない場合）
	if file != nil && fileHeader != nil {
		// ファイル拡張子をチェック
		fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))

		// PDEファイルの確認
		if fileExt != ".pde" {
			return nil, fmt.Errorf("アップロードされたファイルはPDEファイルである必要があります")
		}

		// ファイルデータを読み込む
		fileData, err = io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("ファイルの読み込みに失敗しました: %v", err)
		}

		// PDEファイル用の設定
		fileType = "text/plain"
		fileName = fileHeader.Filename
	} else if fileURL == "" {
		// フロントエンドからもファイルがアップロードされておらず、CloudinaryのURLも提供されていない場合
		return nil, errors.New("ファイルデータまたはURLが必要です")
	}

	// 新しい作品を作成
	work := &models.Work{
		Title:             title,
		Description:       description,
		FileData:          fileData, // PDEファイルの場合はデータ、それ以外はnil
		FileType:          fileType,
		FileName:          fileName,
		FileURL:           fileURL,      // Cloudinaryの場合はURL
		FilePublicID:      filePublicID, // Cloudinaryの場合はpublic_id
		ThumbnailType:     thumbnailType,
		ThumbnailURL:      thumbnailURL,
		ThumbnailPublicID: thumbnailPublicID,
		CodeShared:        codeShared,
		CodeContent:       codeContent,
		UserID:            userID,
		IsGuest:           isGuest,
		GuestNickname:     guestNickname,
	}

	// データベースに保存
	if err := s.workRepo.Create(work); err != nil {
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

	// Processing作品の場合、変換情報を登録
	if file != nil && fileHeader != nil && strings.ToLower(filepath.Ext(fileHeader.Filename)) == ".pde" {
		canvasID := fmt.Sprintf("processingCanvas_%s", utils.GenerateRandomString(8))

		// PDEコードがなければファイルから読み込む
		if codeContent == "" {
			codeContent = string(fileData)
		}

		processingID, err := s.processingRepo.Create(&models.ProcessingWork{
			WorkID:       work.ID,
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

// isAllowedExtension 許可された拡張子かチェック
func (s *workService) isAllowedExtension(ext string) bool {
	allowedExts := []string{".pde", ".png", ".jpg", ".jpeg", ".gif", ".webp"}
	for _, allowed := range allowedExts {
		if strings.EqualFold(ext, allowed) {
			return true
		}
	}
	return false
}

// isImageExtension 画像拡張子かどうかを判定
func (s *workService) isImageExtension(ext string) bool {
	imageExts := []string{".png", ".jpg", ".jpeg", ".gif", ".webp"}
	for _, imgExt := range imageExts {
		if strings.EqualFold(ext, imgExt) {
			return true
		}
	}
	return false
}

// getContentType 拡張子からContent-Typeを取得
func (s *workService) getContentType(ext string) string {
	switch strings.ToLower(ext) {
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

	// 新しいファイルがあるか確認
	var newFilePublicID, newFileURL string
	var newThumbnailPublicID, newThumbnailURL string

	// ファイルがアップロードされた場合は更新
	if file != nil && fileHeader != nil {
		// ファイル拡張子をチェック
		fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if !s.isAllowedExtension(fileExt) {
			return nil, fmt.Errorf("拡張子 %s は許可されていません", fileExt)
		}

		// 画像ファイルの場合は Cloudinary にアップロード
		if s.isImageExtension(fileExt) {
			if _, err := file.Seek(0, 0); err != nil {
				return nil, fmt.Errorf("ファイルの読み込み準備に失敗しました: %v", err)
			}

			// Cloudinaryにアップロード
			newFilePublicID, newFileURL, err = s.cloudinaryService.UploadImage(
				file,
				utils.GenerateRandomString(8)+"_"+fileHeader.Filename,
				75)
			if err != nil {
				return nil, fmt.Errorf("画像のアップロードに失敗しました: %v", err)
			}

			// 古い画像があれば削除予定としてマーク
			oldFilePublicID := work.FilePublicID

			// 新しいURLを設定
			work.FileData = nil
			work.FileURL = newFileURL
			work.FilePublicID = newFilePublicID
			work.FileType = s.getContentType(fileExt)
			work.FileName = fileHeader.Filename

			// 古い画像があれば削除
			if oldFilePublicID != "" {
				_ = s.cloudinaryService.DeleteImage(oldFilePublicID)
			}
		} else {
			// 画像以外はDB に保存
			fileData, err := io.ReadAll(file)
			if err != nil {
				return nil, fmt.Errorf("ファイルの読み込みに失敗しました: %v", err)
			}

			// 古い画像があれば削除
			if work.FilePublicID != "" {
				_ = s.cloudinaryService.DeleteImage(work.FilePublicID)
				work.FilePublicID = ""
				work.FileURL = ""
			}

			// ファイルデータを更新
			work.FileData = fileData
			work.FileType = s.getContentType(fileExt)
			work.FileName = fileHeader.Filename
		}

		// Processing作品の場合、変換情報を更新
		if fileExt == ".pde" {
			// PDEコードがなければファイルから読み込む
			if codeContent == "" {
				if len(work.FileData) > 0 {
					codeContent = string(work.FileData)
				}
			}

			processingWork, err := s.processingRepo.FindByWorkID(id)
			if err != nil {
				// 既存のProcessing情報がない場合は新規作成
				canvasID := fmt.Sprintf("processingCanvas_%s", utils.GenerateRandomString(8))

				processingID, err := s.processingRepo.Create(&models.ProcessingWork{
					WorkID:       work.ID,
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
			} else {
				// 既存のProcessing情報を更新
				processingWork.OriginalName = fileHeader.Filename
				processingWork.PDEContent = codeContent
				processingWork.Status = "pending"

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

		if _, err := thumbnail.Seek(0, 0); err != nil {
			// 新しくアップロードしたファイルがあれば削除
			if newFilePublicID != "" {
				_ = s.cloudinaryService.DeleteImage(newFilePublicID)
			}
			return nil, fmt.Errorf("サムネイルの読み込み準備に失敗しました: %v", err)
		}

		// Cloudinaryにアップロード
		newThumbnailPublicID, newThumbnailURL, err = s.cloudinaryService.UploadImage(
			thumbnail,
			utils.GenerateRandomString(8)+"_thumb_"+thumbnailHeader.Filename,
			70)
		if err != nil {
			// 新しくアップロードしたファイルがあれば削除
			if newFilePublicID != "" {
				_ = s.cloudinaryService.DeleteImage(newFilePublicID)
			}
			return nil, fmt.Errorf("サムネイルのアップロードに失敗しました: %v", err)
		}

		// 古いサムネイルがあれば削除予定としてマーク
		oldThumbnailPublicID := work.ThumbnailPublicID

		// 新しいURLを設定
		work.ThumbnailData = nil
		work.ThumbnailURL = newThumbnailURL
		work.ThumbnailPublicID = newThumbnailPublicID
		work.ThumbnailType = s.getContentType(thumbnailExt)

		// 古いサムネイルがあれば削除
		if oldThumbnailPublicID != "" {
			_ = s.cloudinaryService.DeleteImage(oldThumbnailPublicID)
		}
	}

	// データベースを更新
	if err := s.workRepo.Update(work); err != nil {
		// エラーが発生した場合、新たにアップロードした画像を削除
		if newFilePublicID != "" {
			_ = s.cloudinaryService.DeleteImage(newFilePublicID)
		}
		if newThumbnailPublicID != "" {
			_ = s.cloudinaryService.DeleteImage(newThumbnailPublicID)
		}
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

	// 更新された作品を取得
	return s.GetByID(id)
}

// UpdateWithCloudinary フロントエンドから直接Cloudinaryにアップロードされたファイル情報を使って作品を更新
func (s *workService) UpdateWithCloudinary(
	id uint,
	title, description string,
	file, thumbnail multipart.File,
	fileHeader, thumbnailHeader *multipart.FileHeader,
	fileURL, filePublicID, fileType, fileName,
	thumbnailURL, thumbnailPublicID, thumbnailType string,
	codeShared bool,
	codeContent string,
	tagNames []string,
	userID uint) (*models.Work, error) {

	// 作品を取得
	work, err := s.workRepo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// 権限チェック
	if work.UserID == nil || *work.UserID != userID {
		return nil, errors.New("この作品を更新する権限がありません")
	}

	// 作品の基本情報を更新
	work.Title = title
	work.Description = description
	work.CodeShared = codeShared
	work.CodeContent = codeContent

	// PDEファイルがアップロードされた場合（従来のアップロード方法）
	if file != nil && fileHeader != nil {
		fileExt := strings.ToLower(filepath.Ext(fileHeader.Filename))
		if fileExt != ".pde" {
			return nil, fmt.Errorf("アップロードされたファイルはPDEファイルである必要があります")
		}

		// ファイルデータを読み込む
		fileData, err := io.ReadAll(file)
		if err != nil {
			return nil, fmt.Errorf("ファイルの読み込みに失敗しました: %v", err)
		}

		// 既存のCloudinaryファイルがあれば削除
		if work.FilePublicID != "" {
			if err := s.cloudinaryService.DeleteImage(work.FilePublicID); err != nil {
				fmt.Printf("古い画像を削除できませんでした: %v\n", err)
			}
		}

		// 新しいファイルデータで更新
		work.FileData = fileData
		work.FileType = "text/plain"
		work.FileName = fileHeader.Filename
		work.FileURL = ""
		work.FilePublicID = ""

		// Processing処理を更新または作成
		if fileExt == ".pde" {
			if codeContent == "" {
				codeContent = string(fileData)
			}

			processingWork, err := s.processingRepo.FindByWorkID(id)
			if err != nil {
				// 新規作成
				canvasID := fmt.Sprintf("processingCanvas_%s", utils.GenerateRandomString(8))
				processingID, err := s.processingRepo.Create(&models.ProcessingWork{
					WorkID:       work.ID,
					OriginalName: fileHeader.Filename,
					PDEContent:   codeContent,
					CanvasID:     canvasID,
					Status:       "pending",
				})

				if err != nil {
					fmt.Printf("Processing情報のDBへの保存に失敗しました: %v\n", err)
				} else {
					go func(pid uint) {
						if err := s.lambdaService.InvokePDEConversion(pid); err != nil {
							fmt.Printf("PDE変換処理に失敗しました（処理ID: %d）: %v\n", pid, err)
						}
					}(processingID)
				}
			} else {
				// 既存の情報を更新
				// 既存の情報を更新
				processingWork.OriginalName = fileHeader.Filename
				processingWork.PDEContent = codeContent
				processingWork.Status = "pending"

				if err := s.processingRepo.Update(processingWork); err != nil {
					fmt.Printf("Processing情報の更新に失敗しました: %v\n", err)
				} else {
					go func(pid uint) {
						if err := s.lambdaService.InvokePDEConversion(pid); err != nil {
							fmt.Printf("PDE変換処理に失敗しました（処理ID: %d）: %v\n", pid, err)
						}
					}(processingWork.ID)
				}
			}
		}
	} else if fileURL != "" {
		// フロントエンドでCloudinaryにアップロードした画像がある場合

		// 古いCloudinaryファイルがあれば削除
		if work.FilePublicID != "" && work.FilePublicID != filePublicID {
			if err := s.cloudinaryService.DeleteImage(work.FilePublicID); err != nil {
				fmt.Printf("古い画像を削除できませんでした: %v\n", err)
			}
		}

		// 画像ファイルのデータを更新
		work.FileData = nil
		work.FileURL = fileURL
		work.FilePublicID = filePublicID
		work.FileType = fileType
		work.FileName = fileName
	}

	// サムネイルの更新
	if thumbnailURL != "" {
		// フロントエンドでCloudinaryにアップロードしたサムネイルがある場合

		// 古いCloudinaryサムネイルがあれば削除
		if work.ThumbnailPublicID != "" && work.ThumbnailPublicID != thumbnailPublicID {
			if err := s.cloudinaryService.DeleteImage(work.ThumbnailPublicID); err != nil {
				fmt.Printf("古いサムネイルを削除できませんでした: %v\n", err)
			}
		}

		// サムネイルデータを更新
		work.ThumbnailData = nil
		work.ThumbnailURL = thumbnailURL
		work.ThumbnailPublicID = thumbnailPublicID
		work.ThumbnailType = thumbnailType
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

	// Cloudinaryから画像を削除
	if work.FilePublicID != "" {
		if err := s.cloudinaryService.DeleteImage(work.FilePublicID); err != nil {
			fmt.Printf("画像の削除に失敗しました: %v\n", err)
		}
	}

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

// GetFileData ファイルデータを取得 (従来の方法、互換性のため)
func (s *workService) GetFileData(id uint) ([]byte, string, string, error) {
	return s.workRepo.GetFileData(id)
}

// GetThumbnailData サムネイルデータを取得 (従来の方法、互換性のため)
func (s *workService) GetThumbnailData(id uint) ([]byte, string, error) {
	return s.workRepo.GetThumbnailData(id)
}

// CreatePreview プレビューを作成
func (s *workService) CreatePreview(file multipart.File, fileHeader *multipart.FileHeader, code string) ([]byte, string, error) {
	// ファイルかコードのいずれかが必要
	if file == nil && code == "" {
		return nil, "", errors.New("ファイルまたはコードが必要です")
	}

	var fileData []byte
	var err error
	var contentType string

	if file != nil {
		// ファイルデータを読み込み
		fileData, err = io.ReadAll(file)
		if err != nil {
			return nil, "", fmt.Errorf("ファイルの読み込みに失敗しました: %v", err)
		}
		contentType = s.getContentType(strings.ToLower(filepath.Ext(fileHeader.Filename)))
	} else if code != "" {
		// コードをバイトに変換
		fileData = []byte(code)
		contentType = "text/plain"
	}

	return fileData, contentType, nil
}

// GetProcessingWorkByWorkID 作品IDからProcessing作品情報を取得
func (s *workService) GetProcessingWorkByWorkID(workID uint) (*models.ProcessingWork, error) {
	return s.processingRepo.FindByWorkID(workID)
}
