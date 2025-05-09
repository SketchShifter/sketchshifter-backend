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
	pdeContent := ctx.PostForm("pde_content")
	codeSharedStr := ctx.PostForm("code_shared")
	tagsStr := ctx.PostForm("tags")

	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

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

	// サムネイル画像を取得
	var thumbnail, _ = ctx.FormFile("thumbnail")
	var thumbnailFile multipart.File
	var err error
	if thumbnail != nil {
		thumbnailFile, err = thumbnail.Open()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "サムネイル画像の読み込みに失敗しました"})
			return
		}
		defer thumbnailFile.Close()
	}

	// 作品を作成
	work, err := c.workService.Create(
		title,
		description,
		pdeContent,
		thumbnailFile,
		thumbnail,
		codeShared,
		tags,
		u.ID,
	)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{"work": work})
}

// GetByID IDで作品を取得
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

	ctx.JSON(http.StatusOK, gin.H{"work": work})
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
	pdeContent := ctx.PostForm("pde_content")
	codeSharedStr := ctx.PostForm("code_shared")
	tagsStr := ctx.PostForm("tags")

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

	// サムネイル画像を取得
	var thumbnail, _ = ctx.FormFile("thumbnail")
	var thumbnailFile multipart.File
	if thumbnail != nil {
		thumbnailFile, err = thumbnail.Open()
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "サムネイル画像の読み込みに失敗しました"})
			return
		}
		defer thumbnailFile.Close()
	}

	// 作品を更新
	work, err := c.workService.Update(
		uint(id),
		u.ID,
		title,
		description,
		pdeContent,
		thumbnailFile,
		thumbnail,
		codeShared,
		tags,
	)
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

// GetUserWorks ユーザーの作品一覧を取得
func (c *WorkController) GetUserWorks(ctx *gin.Context) {
	// ユーザーIDを解析
	userID, err := strconv.ParseUint(ctx.Param("userID"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なユーザーIDです"})
		return
	}

	// クエリパラメータを取得
	pageStr := ctx.DefaultQuery("page", "1")
	limitStr := ctx.DefaultQuery("limit", "20")

	// 数値パラメータを解析
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// 作品一覧を取得
	works, total, pages, err := c.workService.GetUserWorks(uint(userID), page, limit)
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
