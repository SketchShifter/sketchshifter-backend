// main.go の修正
package main

import (
	"log"
	"os"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/routes"
	"github.com/gin-gonic/gin"
)

func main() {
	// ログ設定を変更
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("サーバーを起動しています...")

	// 設定をロード
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// Gin モードの設定（環境変数が設定されていない場合はデバッグモード）
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode(gin.DebugMode)
	}

	// カスタムログフォーマットを設定
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		log.Printf("エンドポイント登録: %s %s -> %s (%d handlers)\n", httpMethod, absolutePath, handlerName, nuHandlers)
	}

	// データベース接続
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}

	// SQLDBインスタンスを取得してログ設定を表示
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("SQLDBインスタンス取得に失敗しました: %v", err)
	}
	log.Printf("データベース設定: MaxOpenConns=%d, MaxIdleConns=%d\n",
		sqlDB.Stats().MaxOpenConnections, sqlDB.Stats().Idle)

	// ルーターをセットアップ
	router := routes.SetupRouter(cfg, db)

	// サーバー起動
	log.Printf("サーバーを開始しています... PORT: %s", cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}

// package main

// import (
// 	"log"

// 	"github.com/SketchShifter/sketchshifter_backend/internal/config"
// 	"github.com/SketchShifter/sketchshifter_backend/internal/routes"
// )

// func main() {
// 	// 設定をロード
// 	cfg, err := config.Load()
// 	if err != nil {
// 		log.Fatalf("設定の読み込みに失敗しました: %v", err)
// 	}

// 	// データベース接続
// 	db, err := config.InitDB(cfg)
// 	if err != nil {
// 		log.Fatalf("データベース接続に失敗しました: %v", err)
// 	}

// 	// ルーターをセットアップ
// 	router := routes.SetupRouter(cfg, db)

// 	// サーバー起動
// 	log.Printf("サーバーを開始しています... PORT: %s", cfg.Server.Port)
// 	if err := router.Run(":" + cfg.Server.Port); err != nil {
// 		log.Fatalf("サーバーの起動に失敗しました: %v", err)
// 	}
// }
