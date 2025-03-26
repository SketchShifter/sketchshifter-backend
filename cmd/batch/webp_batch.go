package main

import (
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	defaultBatchSize        = 20
	defaultPendingThreshold = 100
)

func main() {
	// コマンドライン引数の解析
	batchSize := flag.Int("batch-size", defaultBatchSize, "一度に処理する画像数")
	pendingThreshold := flag.Int("threshold", defaultPendingThreshold, "このしきい値を超えると処理が開始される未処理画像の数")
	forceSend := flag.Bool("force", false, "しきい値に関係なく処理を強制的に実行")
	flag.Parse()

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

	// リポジトリの初期化
	imageRepo := repository.NewImageRepository(db)

	// AWSセッションの初期化
	awsSession, err := session.NewSession(&aws.Config{
		Region: aws.String(cfg.AWS.Region),
	})
	if err != nil {
		log.Fatalf("AWSセッションの初期化に失敗しました: %v", err)
	}

	// 未処理の画像数をカウント
	pendingCount, err := imageRepo.CountPendingImages()
	if err != nil {
		log.Fatalf("未処理画像のカウントに失敗しました: %v", err)
	}

	log.Printf("未処理の画像が %d 件見つかりました", pendingCount)

	// しきい値を下回っていて強制実行でない場合は終了
	if pendingCount < int64(*pendingThreshold) && !*forceSend {
		log.Printf("未処理画像数がしきい値 %d を下回っているため、バッチ処理をスキップします", *pendingThreshold)
		return
	}

	// バッチ処理の実行
	log.Printf("バッチ処理を開始します (最大 %d 件)", *batchSize)
	if err := sendBatchToSQS(awsSession, cfg, *batchSize); err != nil {
		log.Fatalf("バッチ処理の送信に失敗しました: %v", err)
	}

	log.Println("バッチ処理が正常に送信されました")
}

// sendBatchToSQS バッチ処理メッセージをSQSに送信
func sendBatchToSQS(awsSession *session.Session, cfg *config.Config, batchSize int) error {
	// SQSクライアントを初期化
	sqsSvc := sqs.New(awsSession)

	// メッセージ内容を作成
	messageBody := struct {
		Type      string `json:"type"`
		BatchSize int    `json:"batchSize"`
		Timestamp string `json:"timestamp"`
	}{
		Type:      "batch_conversion",
		BatchSize: batchSize,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	messageJSON, err := json.Marshal(messageBody)
	if err != nil {
		return err
	}

	// SQSにメッセージを送信
	_, err = sqsSvc.SendMessage(&sqs.SendMessageInput{
		QueueUrl:    aws.String(cfg.AWS.WebpConversionQueueURL),
		MessageBody: aws.String(string(messageJSON)),
	})

	return err
}
