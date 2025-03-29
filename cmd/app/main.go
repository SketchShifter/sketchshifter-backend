package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
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

	// アップロードディレクトリが存在することを確認
	if err := os.MkdirAll(cfg.Storage.UploadDir, 0755); err != nil {
		log.Fatalf("アップロードディレクトリの作成に失敗しました: %v", err)
	}

	// プレビューディレクトリも作成
	previewDir := filepath.Join(cfg.Storage.UploadDir, "preview")
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		log.Printf("プレビューディレクトリの作成に失敗しました: %v", err)
		// 致命的ではないので続行
	}

	// コマンドライン引数をチェック
	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		// マイグレーションモードで実行
		handleMigration(cfg, os.Args[2:])
		return
	}

	// Gin モードの設定（環境変数が設定されていない場合はデバッグモード）
	ginMode := os.Getenv("GIN_MODE")
	if ginMode == "" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(ginMode)
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
	log.Printf("サーバーを開始しています... PORT: %s, MODE: %s", cfg.Server.Port, gin.Mode())
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("サーバーの起動に失敗しました: %v", err)
	}
}

// マイグレーション処理を実行
func handleMigration(cfg *config.Config, args []string) {
	if len(args) == 0 {
		log.Fatal("使用方法: app migrate [up|down]")
	}

	command := args[0]

	// データベース接続
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}

	switch strings.ToLower(command) {
	case "up":
		// マイグレーションを実行
		fmt.Println("マイグレーションを実行中...")
		err = db.AutoMigrate(
			&models.User{},
			&models.ExternalAccount{},
			&models.Tag{},
			&models.Work{},
			&models.Like{},
			&models.Comment{},
			&models.Image{},
			&models.ProcessingWork{},
		)
		if err != nil {
			log.Fatalf("マイグレーションに失敗しました: %v", err)
		}
		fmt.Println("マイグレーションが成功しました")

	case "down":
		// テーブルを削除（逆順）
		fmt.Println("マイグレーションをロールバック中...")
		err = db.Migrator().DropTable(
			&models.Comment{},
			&models.Like{},
			"work_tags",
			&models.Work{},
			&models.Tag{},
			&models.ExternalAccount{},
			&models.User{},
			&models.Image{},
			&models.ProcessingWork{},
		)
		if err != nil {
			log.Fatalf("テーブル削除に失敗しました: %v", err)
		}
		fmt.Println("テーブルの削除が成功しました")

	default:
		log.Fatalf("不明なコマンドです: %s", command)
	}
}
