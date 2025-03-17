package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"
)

// AuthRequired は認証が必要なエンドポイントに使用するミドルウェア
func AuthRequired() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		
		// Bearerトークンの形式を確認
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}
		
		// トークン部分を取得
		token := strings.TrimPrefix(auth, "Bearer ")
		
		// JWTトークンを検証
		userID, err := utils.ValidateJWT(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}
		
		// ユーザーIDをコンテキストに設定
		c.Set("userID", userID)
		
		c.Next()
	}
}
