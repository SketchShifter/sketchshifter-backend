package routes

import (
	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/controllers"
	"github.com/SketchShifter/sketchshifter_backend/internal/middlewares"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
	"github.com/SketchShifter/sketchshifter_backend/internal/services"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRouter ルーターを設定
func SetupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	// Ginルーターを作成
	r := gin.Default()

	// ミドルウェアを設定
	r.Use(middlewares.ErrorMiddleware())
	r.Use(middlewares.CORSMiddleware())

	// 静的ファイルを提供
	r.Static("/uploads", cfg.Storage.UploadDir)

	// リポジトリを作成
	userRepo := repository.NewUserRepository(db)
	workRepo := repository.NewWorkRepository(db)
	tagRepo := repository.NewTagRepository(db)
	commentRepo := repository.NewCommentRepository(db)

	// ユーティリティを作成
	fileUtils := utils.NewFileUtils("/uploads")

	// サービスを作成
	authService := services.NewAuthService(userRepo, cfg)
	workService := services.NewWorkService(workRepo, tagRepo, cfg, fileUtils)
	tagService := services.NewTagService(tagRepo)
	commentService := services.NewCommentService(commentRepo, workRepo)
	userService := services.NewUserService(userRepo, workRepo)

	// コントローラーを作成
	authController := controllers.NewAuthController(authService)
	workController := controllers.NewWorkController(workService)
	tagController := controllers.NewTagController(tagService)
	commentController := controllers.NewCommentController(commentService)
	userController := controllers.NewUserController(userService)
	healthController := controllers.NewHealthController()

	// 認証ミドルウェア
	authMiddleware := middlewares.AuthMiddleware(authService)
	optionalAuthMiddleware := middlewares.OptionalAuthMiddleware(authService)

	// ヘルスチェックエンドポイント（認証不要）
	r.GET("/api/v1/health", healthController.Check)

	// APIグループを作成
	api := r.Group("/api/v1")
	{
		// 認証ルート
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
			auth.POST("/oauth", authController.OAuth)
			auth.GET("/me", authMiddleware, authController.GetMe)
			// 新規追加したエンドポイント
			auth.POST("/change-password", authMiddleware, authController.ChangePassword)
		}

		// 作品ルート
		works := api.Group("/works")
		{
			// 認証不要
			works.GET("", optionalAuthMiddleware, workController.List)
			works.GET("/:id", optionalAuthMiddleware, workController.GetByID)
			works.POST("/preview", workController.CreatePreview)

			// コメント関連
			works.GET("/:id/comments", commentController.List)
			works.POST("/:id/comments", optionalAuthMiddleware, commentController.Create)

			// 認証が必要
			works.POST("", authMiddleware, workController.Create)
			works.PUT("/:id", authMiddleware, workController.Update)
			works.DELETE("/:id", authMiddleware, workController.Delete)
			works.POST("/:id/like", authMiddleware, workController.AddLike)
			works.DELETE("/:id/like", authMiddleware, workController.RemoveLike)
		}

		// コメントルート
		comments := api.Group("/comments")
		{
			comments.PUT("/:id", authMiddleware, commentController.Update)
			comments.DELETE("/:id", authMiddleware, commentController.Delete)
		}

		// タグルート
		api.GET("/tags", tagController.List)

		// ユーザールート
		users := api.Group("/users")
		{
			users.GET("/:id", userController.GetByID)
			users.GET("/:id/works", userController.GetUserWorks)
			users.GET("/favorites", authMiddleware, userController.GetUserFavorites)
			
			// 新規追加したエンドポイント
			users.GET("/me", authMiddleware, userController.GetMe)
			users.PUT("/profile", authMiddleware, userController.UpdateProfile)
			users.GET("/me/works", authMiddleware, userController.GetMyWorks)
		}
	}

	return r
}