package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/lambda"
)

// LambdaService Lambda関数との通信を管理するサービス
type LambdaService interface {
	// PDEをJavaScriptに変換するLambdaを呼び出す
	ConvertPDEToJS(pdeContent string) (string, error)
}

// lambdaService LambdaServiceの実装
type lambdaService struct {
	config       *config.Config
	lambdaClient *lambda.Lambda
}

// NewLambdaService LambdaServiceを作成
func NewLambdaService(cfg *config.Config) LambdaService {
	// AWS セッション作成
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(cfg.Lambda.Region),
	}))

	// Lambda クライアント作成
	lambdaClient := lambda.New(sess)

	return &lambdaService{
		config:       cfg,
		lambdaClient: lambdaClient,
	}
}

// PDEConversionRequest Lambda関数に送るリクエスト構造体
type PDEConversionRequest struct {
	PDEContent string `json:"pdeContent"`
	CanvasID   string `json:"canvasId"`
}

// PDEConversionResponse Lambda関数からのレスポンス構造体
type PDEConversionResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message,omitempty"`
	JSContent string `json:"jsContent,omitempty"`
}

// ConvertPDEToJS PDEをJavaScriptに変換するLambdaを呼び出す
func (s *lambdaService) ConvertPDEToJS(pdeContent string) (string, error) {
	if pdeContent == "" {
		return "", fmt.Errorf("PDEコンテンツが空です")
	}

	// CanvasID生成（一意な識別子）
	canvasID := fmt.Sprintf("canvas_%d", time.Now().UnixNano())

	// Lambda関数のパラメータを作成
	requestPayload := PDEConversionRequest{
		PDEContent: pdeContent,
		CanvasID:   canvasID,
	}

	// JSONに変換
	payload, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("リクエストのJSONエンコードに失敗しました: %v", err)
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
		return "", fmt.Errorf("Lambda関数の呼び出しに失敗しました: %v", err)
	}

	// レスポンスをパース
	var lambdaResponse PDEConversionResponse
	if err := json.Unmarshal(output.Payload, &lambdaResponse); err != nil {
		return "", fmt.Errorf("Lambda関数のレスポンスをパースできませんでした: %v", err)
	}

	// 処理結果を確認
	if !lambdaResponse.Success {
		return "", fmt.Errorf("PDE変換処理が失敗しました: %s", lambdaResponse.Message)
	}

	// JSコンテンツを確認
	if lambdaResponse.JSContent == "" {
		return "", fmt.Errorf("Lambda関数から空のJSコンテンツが返されました")
	}

	return lambdaResponse.JSContent, nil
}
