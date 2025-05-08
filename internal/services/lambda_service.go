package services

import (
	"encoding/json"
	"fmt"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

// LambdaService Lambda関数との通信を管理するサービス
type LambdaService interface {
	// PDEをJavaScriptに変換するLambdaを呼び出す
	InvokePDEConversion(processingID uint) error
}

// lambdaService LambdaServiceの実装
type lambdaService struct {
	config         *config.Config
	processingRepo repository.ProcessingRepository
	lambdaClient   *lambda.Lambda
}

// NewLambdaService LambdaServiceを作成
func NewLambdaService(cfg *config.Config, processingRepo repository.ProcessingRepository) LambdaService {
	// AWS セッション作成
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(cfg.Lambda.Region),
	}))

	// Lambda クライアント作成
	lambdaClient := lambda.New(sess)

	return &lambdaService{
		config:         cfg,
		processingRepo: processingRepo,
		lambdaClient:   lambdaClient,
	}
}

// PDEConversionRequest Lambda関数に送るリクエスト構造体
type PDEConversionRequest struct {
	ProcessingID uint   `json:"processingId"`
	PDEContent   string `json:"pdeContent"`
	FileName     string `json:"fileName"`
	CanvasID     string `json:"canvasId"`
}

// PDEConversionResponse Lambda関数からのレスポンス構造体
type PDEConversionResponse struct {
	Success      bool   `json:"success"`
	Message      string `json:"message,omitempty"`
	ProcessingID uint   `json:"processingId"`
	JSContent    string `json:"jsContent,omitempty"`
}

// InvokePDEConversion PDEをJavaScriptに変換するLambdaを呼び出す
func (s *lambdaService) InvokePDEConversion(processingID uint) error {
	// Processing情報を取得
	processing, err := s.processingRepo.FindByID(processingID)
	if err != nil {
		return fmt.Errorf("Processingデータの取得に失敗しました: %v", err)
	}

	// 処理状態を更新
	processing.Status = "processing"
	processing.ErrorMessage = ""
	if err := s.processingRepo.Update(processing); err != nil {
		return fmt.Errorf("処理状態の更新に失敗しました: %v", err)
	}

	// PDEのコンテンツを取得
	pdeContent := processing.PDEContent
	if pdeContent == "" {
		// コンテンツが空の場合はエラー
		processing.Status = "error"
		processing.ErrorMessage = "PDEコンテンツが見つかりません"
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("PDEコンテンツが見つかりません")
	}

	// Lambda関数のパラメータを作成
	requestPayload := PDEConversionRequest{
		ProcessingID: processingID,
		PDEContent:   pdeContent,
		FileName:     processing.OriginalName,
		CanvasID:     processing.CanvasID,
	}

	// JSONに変換
	payload, err := json.Marshal(requestPayload)
	if err != nil {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("リクエストのJSONエンコードに失敗しました: %v", err)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("リクエストのJSONエンコードに失敗しました: %v", err)
	}

	// Lambda関数を呼び出し
	input := &lambda.InvokeInput{
		FunctionName:   aws.String(s.config.Lambda.FunctionName),
		Payload:        payload,
		InvocationType: aws.String("RequestResponse"), // 同期呼び出し
	}

	// Lambda呼び出し実行
	output, err := s.lambdaClient.Invoke(input)
	if err != nil {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("Lambda関数の呼び出しに失敗しました: %v", err)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("Lambda関数の呼び出しに失敗しました: %v", err)
	}

	// レスポンスをパース
	var lambdaResponse PDEConversionResponse
	if err := json.Unmarshal(output.Payload, &lambdaResponse); err != nil {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("Lambda関数のレスポンスをパースできませんでした: %v", err)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("Lambda関数のレスポンスをパースできませんでした: %v", err)
	}

	// 処理結果を確認
	if !lambdaResponse.Success {
		processing.Status = "error"
		processing.ErrorMessage = lambdaResponse.Message
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("PDE変換処理が失敗しました: %s", lambdaResponse.Message)
	}

	// JSコンテンツを確認
	if lambdaResponse.JSContent == "" {
		processing.Status = "error"
		processing.ErrorMessage = "Lambda関数から空のJSコンテンツが返されました"
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("Lambda関数から空のJSコンテンツが返されました")
	}

	// 処理成功を記録
	processing.Status = "processed"
	processing.JSContent = lambdaResponse.JSContent
	processing.ErrorMessage = ""

	if err := s.processingRepo.Update(processing); err != nil {
		return fmt.Errorf("処理結果の更新に失敗しました: %v", err)
	}

	return nil
}
