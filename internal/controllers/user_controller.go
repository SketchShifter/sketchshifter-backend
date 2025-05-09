package controllers

import (
	"net/http"
	"strconv"

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

	ctx.JSON(http.StatusOK, user)
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
	var req struct {
		Name     string `json:"name"`
		Nickname string `json:"nickname"`
		Bio      string `json:"bio"`
	}
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
