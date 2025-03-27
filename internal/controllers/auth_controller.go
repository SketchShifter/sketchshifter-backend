package controllers

import (
	"net/http"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/models"
	"github.com/SketchShifter/sketchshifter_backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthController 認証に関するコントローラー
type AuthController struct {
	authService services.AuthService
}

// NewAuthController AuthControllerを作成
func NewAuthController(authService services.AuthService) *AuthController {
	return &AuthController{
		authService: authService,
	}
}

// RegisterRequest ユーザー登録リクエスト
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name" binding:"required"`
	Nickname string `json:"nickname" binding:"required"`
}

// LoginRequest ログインリクエスト
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// OAuthRequest OAuth認証リクエスト
type OAuthRequest struct {
	Provider string `json:"provider" binding:"required"`
	Code     string `json:"code" binding:"required"`
}

// AuthResponse 認証レスポンス
type AuthResponse struct {
	User  interface{} `json:"user"`
	Token string      `json:"token"`
}

// Register ユーザー登録
func (c *AuthController) Register(ctx *gin.Context) {
	var req RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := c.authService.Register(req.Email, req.Password, req.Name, req.Nickname)
	if err != nil {
		if strings.Contains(err.Error(), "既に使用されています") {
			ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusCreated, AuthResponse{
		User:  user,
		Token: token,
	})
}

// Login ログイン
func (c *AuthController) Login(ctx *gin.Context) {
	var req LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := c.authService.Login(req.Email, req.Password)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, AuthResponse{
		User:  user,
		Token: token,
	})
}

// OAuth OAuth認証
func (c *AuthController) OAuth(ctx *gin.Context) {
	var req OAuthRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, token, err := c.authService.OAuth(req.Provider, req.Code)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, AuthResponse{
		User:  user,
		Token: token,
	})
}

// GetMe 現在のユーザー情報を取得
func (c *AuthController) GetMe(ctx *gin.Context) {
	// コンテキストからユーザーを取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}

	ctx.JSON(http.StatusOK, user)
}

// auth_controller.go に追加する部分

// PasswordChangeRequest パスワード変更リクエスト
type PasswordChangeRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
}

// ChangePassword パスワードを変更
func (c *AuthController) ChangePassword(ctx *gin.Context) {
	// ユーザーを取得
	user, exists := ctx.Get("user")
	if !exists {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
		return
	}
	u := user.(*models.User)

	// リクエストをバインド
	var req PasswordChangeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// パスワードを変更
	if err := c.authService.ChangePassword(u.ID, req.CurrentPassword, req.NewPassword); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "パスワードが正常に変更されました"})
}

// GetMe、UpdateProfile、GetMyWorksメソッドを追加
// GetMe 自分のユーザー情報を取得
// func (c *UserController) GetMe(ctx *gin.Context) {
// 	// ユーザー情報を取得
// 	user, exists := ctx.Get("user")
// 	if !exists {
// 		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, user)
// }

// // UpdateProfile 自分のプロフィールを更新
// func (c *UserController) UpdateProfile(ctx *gin.Context) {
// 	// ユーザー情報を取得
// 	user, exists := ctx.Get("user")
// 	if !exists {
// 		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
// 		return
// 	}
// 	u := user.(*models.User)

// 	// リクエストをバインド
// 	var req UserUpdateRequest
// 	if err := ctx.ShouldBindJSON(&req); err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	// プロフィールを更新
// 	updatedUser, err := c.userService.UpdateProfile(u.ID, req.Name, req.Nickname, req.Bio)
// 	if err != nil {
// 		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, updatedUser)
// }

// // GetMyWorks 自分の作品一覧を取得
// func (c *UserController) GetMyWorks(ctx *gin.Context) {
// 	// ユーザー情報を取得
// 	user, exists := ctx.Get("user")
// 	if !exists {
// 		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
// 		return
// 	}
// 	u := user.(*models.User)

// 	// クエリパラメータを取得
// 	pageStr := ctx.DefaultQuery("page", "1")
// 	limitStr := ctx.DefaultQuery("limit", "20")

// 	// 数値パラメータを解析
// 	page, err := strconv.Atoi(pageStr)
// 	if err != nil || page < 1 {
// 		page = 1
// 	}

// 	limit, err := strconv.Atoi(limitStr)
// 	if err != nil || limit < 1 || limit > 100 {
// 		limit = 20
// 	}

// 	// 作品一覧を取得
// 	works, total, pages, err := c.userService.GetUserWorks(u.ID, page, limit)
// 	if err != nil {
// 		ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	ctx.JSON(http.StatusOK, WorksResponse{
// 		Works: works,
// 		Total: total,
// 		Pages: pages,
// 		Page:  page,
// 	})
// }
