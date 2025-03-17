package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/SketchShifter/sketchshifter_backend/internal/controllers"
	"github.com/SketchShifter/sketchshifter_backend/internal/middlewares"
)

// SetupRoutes はルーティングを設定する
func SetupRoutes(r *gin.Engine) {
	// CORSミドルウェアの設定
	r.Use(middlewares.CORS())

	// APIグループ
	api := r.Group("/api")
	{
		// 認証エンドポイント
		auth := api.Group("/auth")
		{
			auth.POST("/register", controllers.Register)
			auth.POST("/login", controllers.Login)
			auth.GET("/me", middlewares.AuthRequired(), controllers.GetCurrentUser)
		}

		// 作品エンドポイント
		works := api.Group("/works")
		{
			works.GET("", controllers.GetWorks)
			works.GET("/:id", controllers.GetWork)
			works.POST("", middlewares.AuthRequired(), controllers.CreateWork)
			works.PUT("/:id", middlewares.AuthRequired(), controllers.UpdateWork)
			works.DELETE("/:id", middlewares.AuthRequired(), controllers.DeleteWork)
			works.POST("/preview", controllers.PreviewWork)
			
			// いいね
			works.POST("/:id/like", middlewares.AuthRequired(), controllers.LikeWork)
			works.DELETE("/:id/like", middlewares.AuthRequired(), controllers.UnlikeWork)
			
			// お気に入り
			works.POST("/:id/favorite", middlewares.AuthRequired(), controllers.FavoriteWork)
			works.DELETE("/:id/favorite", middlewares.AuthRequired(), controllers.UnfavoriteWork)
			
			// コメント
			works.GET("/:id/comments", controllers.GetWorkComments)
			works.POST("/:id/comments", controllers.CreateComment)
		}

		// コメント管理
		comments := api.Group("/comments")
		{
			comments.PUT("/:id", middlewares.AuthRequired(), controllers.UpdateComment)
			comments.DELETE("/:id", middlewares.AuthRequired(), controllers.DeleteComment)
		}

		// タグ
		tags := api.Group("/tags")
		{
			tags.GET("", controllers.GetTags)
		}

		// ユーザー
		users := api.Group("/users")
		{
			users.GET("/favorites", middlewares.AuthRequired(), controllers.GetUserFavorites)
			users.GET("/:id", controllers.GetUser)
			users.GET("/:id/works", controllers.GetUserWorks)
		}
	}
}
