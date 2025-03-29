package routes

import (
	"path/filepath"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/controllers"
	"github.com/SketchShifter/sketchshifter_backend/internal/middlewares"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
	"github.com/SketchShifter/sketchshifter_backend/internal/services"

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

	// 静的ファイル提供の設定
	r.Static("/uploads", cfg.Storage.UploadDir)

	// 直接assetsディレクトリを公開
	assetsDir := filepath.Join(cfg.Storage.UploadDir, "assets")
	r.Static("/assets", assetsDir)

	// リポジトリを作成
	userRepo := repository.NewUserRepository(db)
	workRepo := repository.NewWorkRepository(db)
	tagRepo := repository.NewTagRepository(db)
	commentRepo := repository.NewCommentRepository(db)
	processingRepo := repository.NewProcessingRepository(db)

	// サービスを作成
	authService := services.NewAuthService(userRepo, cfg)
	fileService := services.NewFileService(cfg)
	lambdaService := services.NewLambdaService(cfg, processingRepo)
	workService := services.NewWorkService(
		workRepo,
		tagRepo,
		processingRepo,
		cfg,
		fileService,
		lambdaService,
	)
	tagService := services.NewTagService(tagRepo)
	commentService := services.NewCommentService(commentRepo, workRepo)
	userService := services.NewUserService(userRepo, workRepo)
	healthService := services.NewHealthService()

	// コントローラーを作成
	authController := controllers.NewAuthController(authService)
	workController := controllers.NewWorkController(workService)
	tagController := controllers.NewTagController(tagService)
	commentController := controllers.NewCommentController(commentService)
	userController := controllers.NewUserController(userService)
	healthController := controllers.NewHealthController(healthService)
	lambdaController := controllers.NewLambdaController(lambdaService)

	// 新しい専用コントローラーを作成
	uploadController := controllers.NewUploadController(
		cfg.Storage.UploadDir,
		cfg.CloudflareWorker.URL,
		cfg.CloudflareWorker.APIKey,
		cfg.CloudflareWorker.Bucket,
	)

	previewController := controllers.NewPreviewController(
		cfg.Storage.UploadDir,
		cfg.AWS.LambdaEndpoint,
		"/uploads/preview",
	)

	// 認証ミドルウェア
	authMiddleware := middlewares.AuthMiddleware(authService)
	optionalAuthMiddleware := middlewares.OptionalAuthMiddleware(authService)

	// APIグループを作成
	api := r.Group("/api/v1")
	{
		// ヘルスチェックルート（認証不要）
		api.GET("/health", healthController.Check)

		// 認証ルート
		auth := api.Group("/auth")
		{
			auth.POST("/register", authController.Register)
			auth.POST("/login", authController.Login)
			auth.POST("/oauth", authController.OAuth)
			auth.GET("/me", authMiddleware, authController.GetMe)
			auth.POST("/change-password", authMiddleware, authController.ChangePassword)
		}

		// 作品ルート
		works := api.Group("/works")
		{
			// 認証不要
			works.GET("", optionalAuthMiddleware, workController.List)
			works.GET("/:id", optionalAuthMiddleware, workController.GetByID)

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

		// ファイルアップロード（新しいコントローラーを使用）
		api.POST("/upload", uploadController.UploadFile)

		// プレビュー生成（新しいコントローラーを使用）
		api.POST("/preview", previewController.CreatePreview)

		// Lambda関連ルート（デバッグ・管理用）
		lambda := api.Group("/lambda")
		{
			// 手動変換起動用
			lambda.POST("/process/:id", lambdaController.ProcessPDE)

			// プレビュー用
			lambda.POST("/preview", previewController.CreatePreview)
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
			users.GET("/me", authMiddleware, userController.GetMe)
			users.PUT("/profile", authMiddleware, userController.UpdateProfile)
			users.GET("/my-works", authMiddleware, userController.GetMyWorks)
		}
	}

	return r
}
