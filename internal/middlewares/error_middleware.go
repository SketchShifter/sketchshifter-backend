package middlewares

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// ErrorMiddleware エラーハンドリングミドルウェア
func ErrorMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// ここでパニックをキャッチしてエラーレスポンスを返す
				debug.PrintStack()
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": "サーバーエラーが発生しました",
				})
			}
		}()
		ctx.Next()
	}
}

// CORSMiddleware CORSミドルウェア
func CORSMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		ctx.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
