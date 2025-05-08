package main

import (
	"fmt"
	"log"
	"os"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/models"
)

func main() {
	// 引数をチェック
	if len(os.Args) < 2 {
		log.Fatal("使用方法: migrate [up|down]")
	}

	// 設定をロード
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}

	// データベース接続
	db, err := config.InitDB(cfg)
	if err != nil {
		log.Fatalf("データベース接続に失敗しました: %v", err)
	}

	command := os.Args[1]

	switch command {
	case "up":
		// マイグレーションを実行
		err = db.AutoMigrate(
			&models.User{},
			&models.Tag{},
			&models.Work{},
			&models.Like{},
			&models.Comment{},
			&models.ProcessingWork{},
		)
		if err != nil {
			log.Fatalf("マイグレーションに失敗しました: %v", err)
		}
		fmt.Println("マイグレーションが成功しました")

	case "down":
		// テーブルを削除（逆順）
		err = db.Migrator().DropTable(
			&models.Comment{},
			&models.Like{},
			"work_tags",
			&models.Work{},
			&models.Tag{},
			&models.User{},
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
