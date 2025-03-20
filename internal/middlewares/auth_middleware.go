package middlewares

import (
	"net/http"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/services"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 認証ミドルウェア
func AuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Authorizationヘッダーを取得
		authHeader := ctx.GetHeader("Authorization")

		// ヘッダーがない場合は認証エラー
		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "認証が必要です"})
			ctx.Abort()
			return
		}

		// Bearer トークンの形式かチェック
		if !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "無効な認証形式です"})
			ctx.Abort()
			return
		}

		// トークンを抽出
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// ユーザーを取得
		user, err := authService.GetUserFromToken(tokenString)
		if err != nil {
			ctx.JSON(http.StatusUnauthorized, gin.H{"error": "無効なトークンです"})
			ctx.Abort()
			return
		}

		// ユーザーをコンテキストに保存
		ctx.Set("user", user)
		ctx.Next()
	}
}

// OptionalAuthMiddleware オプショナル認証ミドルウェア（認証がない場合もエラーを返さない）
func OptionalAuthMiddleware(authService services.AuthService) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Authorizationヘッダーを取得
		authHeader := ctx.GetHeader("Authorization")

		// ヘッダーがない場合は認証なしで続行
		if authHeader == "" {
			ctx.Next()
			return
		}

		// Bearer トークンの形式かチェック
		if !strings.HasPrefix(authHeader, "Bearer ") {
			ctx.Next()
			return
		}

		// トークンを抽出
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// ユーザーを取得
		user, err := authService.GetUserFromToken(tokenString)
		if err != nil {
			ctx.Next()
			return
		}

		// ユーザーをコンテキストに保存
		ctx.Set("user", user)
		ctx.Next()
	}
}
