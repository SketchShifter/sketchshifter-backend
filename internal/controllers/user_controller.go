package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/services"

	"github.com/gin-gonic/gin"
)

// UserController ユーザーに関するコントローラー
type UserController struct {
	userService services.UserService
}

// NewUserController UserControllerを作成
func NewUserController(userService services.UserService) *UserController {
	return &UserController{
		userService: userService,
	}
}

// UserResponse ユーザーレスポンス
type UserResponse struct {
	User interface{} `json:"user"`
}

// UserUpdateRequest ユーザープロフィール更新リクエスト
type UserUpdateRequest struct {
	Name     string `json:"name"`
	Nickname string `json:"nickname"`
	Bio      string `json:"bio"`
}

// GetByID IDでユーザーを取得
func (c *UserController) GetByID(ctx *gin.Context) {
	// IDを解析
	id, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
		return
	}

	// ユーザーを取得
	user, err := c.userService.GetByID(uint(id))
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "ユーザーが見つかりません"})
		return
	}

	ctx.JSON(http.StatusOK, UserResponse{User: user})
}

// GetUserWorks ユーザーの作品一覧を取得
func (c *UserController) GetUserWorks(ctx *gin.Context) {
	// ユーザーIDを解析
	userID, err := strconv.ParseUint(ctx.Param("id"), 10, 32)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なIDです"})
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
	works, total, pages, err := c.userService.GetUserWorks(uint(userID), page, limit)
	if err != nil {
		if strings.Contains(err.Error(), "見つかりません") {
			ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
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

// GetUserFavorites ユーザーのお気に入り作品一覧を取得
func (c *UserController) GetUserFavorites(ctx *gin.Context) {
	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

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

	// お気に入り作品一覧を取得
	works, total, pages, err := c.userService.GetUserFavorites(u.ID, page, limit)
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

// GetMe 自分のユーザー情報を取得
func (c *UserController) GetMe(ctx *gin.Context) {
	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// UpdateProfile 自分のプロフィールを更新
func (c *UserController) UpdateProfile(ctx *gin.Context) {
	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// リクエストをバインド
	var req UserUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// プロフィールを更新
	updatedUser, err := c.userService.UpdateProfile(u.ID, req.Name, req.Nickname, req.Bio)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedUser)
}

// GetMyWorks 自分の作品一覧を取得
func (c *UserController) GetMyWorks(ctx *gin.Context) {
	// ユーザー情報を取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

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
	works, total, pages, err := c.userService.GetUserWorks(u.ID, page, limit)
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
