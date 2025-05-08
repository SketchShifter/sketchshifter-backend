package controllers

import (
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/services"

	"github.com/gin-gonic/gin"
)

// WorkController 作品に関するコントローラー
type WorkController struct {
	workService services.WorkService
}

// NewWorkController WorkControllerを作成
func NewWorkController(workService services.WorkService) *WorkController {
	return &WorkController{
		workService: workService,
	}
}

// Create 新しい作品を作成
func (c *WorkController) Create(ctx *gin.Context) {
	// マルチパートフォームを解析
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "マルチパートフォームの解析に失敗しました"})
		return
	}

	// フォームデータを取得
	title := ctx.PostForm("title")
	description := ctx.PostForm("description")
	codeSharedStr := ctx.PostForm("code_shared")
	codeContent := ctx.PostForm("code_content")
	tagsStr := ctx.PostForm("tags")
	isGuestStr := ctx.DefaultPostForm("is_guest", "false")
	guestNickname := ctx.PostForm("guest_nickname")

	// Cloudinaryから取得したURLとpublic_id（フロントエンドアップロード用）
	fileURL := ctx.PostForm("file_url")
	filePublicID := ctx.PostForm("file_public_id")
	fileType := ctx.PostForm("file_type")
	fileName := ctx.PostForm("file_name")
	thumbnailURL := ctx.PostForm("thumbnail_url")
	thumbnailPublicID := ctx.PostForm("thumbnail_public_id")
	thumbnailType := ctx.PostForm("thumbnail_type")

	// タグを解析
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	// boolean値を解析
	codeShared := codeSharedStr == "true" || codeSharedStr == "1"
	isGuest := isGuestStr == "true" || isGuestStr == "1"

	// ファイル情報の変数準備
	var file multipart.File
	var fileHeader *multipart.FileHeader
	var thumbnail multipart.File
	var thumbnailHeader *multipart.FileHeader
	var err error

	// Cloudinaryを使わずPDEファイルをアップロードする場合のみファイルを取得
	if fileURL == "" {
		file, fileHeader, err = ctx.Request.FormFile("file")
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "ファイルが必要です"})
			return
		}
		defer file.Close()
	}

	// ユーザー情報を取得
	var userID *uint
	if !isGuest {
		user, exists := ctx.Get("user")
		if !exists {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
			return
		}
		u := user.(*models.User)
		userID = &u.ID
	}

	// 作品を作成
	work, err := c.workService.CreateWithCloudinary(
		title,
		description,
		file,
		thumbnail,
		fileHeader,
		thumbnailHeader,
		fileURL,
		filePublicID,
		fileType,
		fileName,
		thumbnailURL,
		thumbnailPublicID,
		thumbnailType,
		codeShared,
		codeContent,
		tags,
		userID,
		isGuest,
		guestNickname,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"work": work})
}

// GetByID IDで作品を取得（processingデータを含める修正版）
func (c *WorkController) GetByID(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// 作品を取得
	work, err := c.workService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "作品が見つかりません"})
		return
	}

	// Processing作品データを取得（PDEファイルの場合）
	var processingWork *models.ProcessingWork
	if strings.HasSuffix(work.FileName, ".pde") || work.FileType == "text/plain" {
		processingWork, _ = c.workService.GetProcessingWorkByWorkID(work.ID)
	}

	// レスポンスを作成
	response := gin.H{"work": work}
	if processingWork != nil {
		response["processing_work"] = processingWork
	}

	ctx.JSON(http.StatusOK, response)
}

// Update 作品を更新
func (c *WorkController) Update(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// マルチパートフォームを解析
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "マルチパートフォームの解析に失敗しました"})
		return
	}

	// フォームデータを取得
	title := ctx.PostForm("title")
	description := ctx.PostForm("description")
	codeSharedStr := ctx.PostForm("code_shared")
	codeContent := ctx.PostForm("code_content")
	tagsStr := ctx.PostForm("tags")

	// Cloudinaryから取得したURLとpublic_id（フロントエンドアップロード用）
	fileURL := ctx.PostForm("file_url")
	filePublicID := ctx.PostForm("file_public_id")
	fileType := ctx.PostForm("file_type")
	fileName := ctx.PostForm("file_name")
	thumbnailURL := ctx.PostForm("thumbnail_url")
	thumbnailPublicID := ctx.PostForm("thumbnail_public_id")
	thumbnailType := ctx.PostForm("thumbnail_type")

	// タグを解析
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
		for i, tag := range tags {
			tags[i] = strings.TrimSpace(tag)
		}
	}

	// boolean値を解析
	codeShared := codeSharedStr == "true" || codeSharedStr == "1"

	// ファイルを取得（オプション）
	var file multipart.File
	var fileHeader *multipart.FileHeader
	if fileURL == "" {
		file, fileHeader, err = ctx.Request.FormFile("file")
		if err == nil && file != nil {
			defer file.Close()
		}
	}

	// 作品を更新
	var work *models.Work
	if fileURL != "" || thumbnailURL != "" {
		// Cloudinaryデータがある場合
		work, err = c.workService.UpdateWithCloudinary(
			uint(id),
			title,
			description,
			file,
			nil, // thumbnail は直接アップロードしない
			fileHeader,
			nil, // thumbnailHeader は直接アップロードしない
			fileURL,
			filePublicID,
			fileType,
			fileName,
			thumbnailURL,
			thumbnailPublicID,
			thumbnailType,
			codeShared,
			codeContent,
			tags,
			u.ID,
		)
	} else {
		// 従来の更新方法
		work, err = c.workService.Update(
			uint(id),
			title,
			description,
			file,
			nil, // thumbnail は直接アップロードしない
			fileHeader,
			nil, // thumbnailHeader は直接アップロードしない
			codeShared,
			codeContent,
			tags,
			u.ID,
		)
	}

	if err != nil {
		if strings.Contains(err.Error(), "権限がありません") {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"work": work})
}

// Delete 作品を削除
func (c *WorkController) Delete(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// 作品を削除
	if err := c.workService.Delete(uint(id), u.ID); err != nil {
		if strings.Contains(err.Error(), "権限がありません") {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.Status(http.StatusNoContent)
}

// List 作品一覧を取得
func (c *WorkController) List(ctx *gin.Context) {
	// クエリパラメータを取得
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "20")
	search := ctx.Query("search")
	tag := ctx.Query("tag")
	userIDStr := ctx.Query("user_id")
	sort := ctx.DefaultQuery("sort", "newest")

	// 数値パラメータを解析
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// ユーザーIDを解析（オプション）
	var userID *uint
	if userIDStr != "" {
		id, err := strconv.ParseUint(userIDStr, 10, 32)
		if err == nil {
			uid := uint(id)
			userID = &uid
		}
	}

	// 作品一覧を取得
	works, total, pages, err := c.workService.List(page, limit, search, tag, userID, sort)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"works": works,
		"total": total,
		"pages": pages,
		"page":  page,
	})
}

// HasLiked ユーザーがいいねしているか確認
func (c *WorkController) HasLiked(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// いいね状態を確認
	liked, err := c.workService.HasLiked(u.ID, uint(id))
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"liked": liked})
}

// AddLike いいねを追加
func (c *WorkController) AddLike(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// いいねを追加
	likesCount, err := c.workService.AddLike(u.ID, uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"likes_count": likesCount,
	})
}

// RemoveLike いいねを削除
func (c *WorkController) RemoveLike(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// いいねを削除
	likesCount, err := c.workService.RemoveLike(u.ID, uint(id))
	if err != nil {
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"likes_count": likesCount,
	})
}

// GetFile ファイルデータを取得
func (c *WorkController) GetFile(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// 作品情報を取得
	work, err := c.workService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "作品が見つかりません"})
		return
	}

	// Cloudinaryの場合はリダイレクト
	if work.FileURL != "" {
		ctx.Redirect(http.StatusFound, work.FileURL)
		return
	}

	// 従来のDB保存ファイル取得（互換性のため）
	fileData, contentType, _, err := c.workService.GetFileData(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "ファイルが見つかりません"})
		return
	}

	ctx.Header("Content-Type", contentType)
	ctx.Header("Cache-Control", "public, max-age=86400") // キャッシュ設定
	ctx.Data(http.StatusOK, contentType, fileData)
}

// GetThumbnail サムネイルデータを取得
func (c *WorkController) GetThumbnail(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// 作品情報を取得
	work, err := c.workService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "作品が見つかりません"})
		return
	}

	// Cloudinaryの場合はリダイレクト
	if work.ThumbnailURL != "" {
		ctx.Redirect(http.StatusFound, work.ThumbnailURL)
		return
	}

	// 従来のDB保存サムネイル取得（互換性のため）
	thumbnailData, contentType, err := c.workService.GetThumbnailData(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "サムネイルが見つかりません"})
		return
	}

	// サムネイルデータがない場合
	if len(thumbnailData) == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "サムネイルが設定されていません"})
		return
	}

	ctx.Header("Content-Type", contentType)
	ctx.Header("Cache-Control", "public, max-age=86400") // キャッシュ設定
	ctx.Data(http.StatusOK, contentType, thumbnailData)
}

// CreatePreview プレビューを作成
func (c *WorkController) CreatePreview(ctx *gin.Context) {
	// マルチパートフォームを解析
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "マルチパートフォームの解析に失敗しました"})
		return
	}

	// ファイルまたはコードを取得
	file, fileHeader, err := ctx.Request.FormFile("file")
	code := ctx.PostForm("code")

	if (err != nil || file == nil) && code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ファイルまたはコードが必要です"})
		return
	}

	if file != nil {
		defer file.Close()
	}

	// プレビューを作成
	previewData, contentType, err := c.workService.CreatePreview(file, fileHeader, code)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// プレビューデータを返す
	ctx.Data(http.StatusOK, contentType, previewData)
}
