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
