package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/repository"
)

// LambdaService Lambda関数との通信を管理するサービス
type LambdaService interface {
	// PDEをJavaScriptに変換するLambdaを呼び出す
	InvokePDEConversion(processingID uint) error

	// プレビュー用のPDE変換を実行
	InvokePreviewConversion(preview *PreviewProcessing) error
}

// lambdaService LambdaServiceの実装
type lambdaService struct {
	config         *config.Config
	processingRepo repository.ProcessingRepository
	httpClient     *http.Client
}

// NewLambdaService LambdaServiceを作成
func NewLambdaService(cfg *config.Config, processingRepo repository.ProcessingRepository) LambdaService {
	return &lambdaService{
		config:         cfg,
		processingRepo: processingRepo,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Lambda呼び出しは最大60秒のタイムアウト
		},
	}
}

// PreviewProcessing プレビュー用の一時的なProcessing構造体
type PreviewProcessing struct {
	ID           uint
	FileName     string
	OriginalName string
	PDEContent   string
	CanvasID     string
}

// PDEConversionRequest Lambda関数に送るリクエスト構造体
type PDEConversionRequest struct {
	ProcessingID uint   `json:"processingId"`
	PDEContent   string `json:"pdeContent"`
	FileName     string `json:"fileName"`
	OriginalName string `json:"originalName"`
	CanvasID     string `json:"canvasId"`
	IsPreview    bool   `json:"isPreview"` // プレビュー用フラグを追加
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
		// コンテンツが空の場合はファイルから読み込む
		if processing.PDEPath != "" {
			// アップロードディレクトリからPDEファイルのパスを構築
			pdePath := filepath.Join(s.config.Storage.UploadDir, strings.TrimPrefix(processing.PDEPath, "/uploads/"))

			// ファイルを読み込み
			content, err := os.ReadFile(pdePath)
			if err != nil {
				// エラー状態を更新
				processing.Status = "error"
				processing.ErrorMessage = fmt.Sprintf("PDEファイルの読み込みに失敗しました: %v", err)
				_ = s.processingRepo.Update(processing)
				return fmt.Errorf("PDEファイルの読み込みに失敗しました: %v", err)
			}

			pdeContent = string(content)

			// コンテンツを保存（将来の処理のため）
			processing.PDEContent = pdeContent
			_ = s.processingRepo.Update(processing)
		} else {
			// PDEコンテンツもPDEパスも無い場合はエラー
			processing.Status = "error"
			processing.ErrorMessage = "PDEコンテンツが見つかりません"
			_ = s.processingRepo.Update(processing)
			return fmt.Errorf("PDEコンテンツが見つかりません")
		}
	}

	// Lambda関数のエンドポイント
	lambdaURL := s.config.AWS.LambdaEndpoint
	if lambdaURL == "" {
		processing.Status = "error"
		processing.ErrorMessage = "Lambda関数のエンドポイントが設定されていません"
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("Lambda関数のエンドポイントが設定されていません")
	}

	// リクエスト構築
	request := PDEConversionRequest{
		ProcessingID: processingID,
		PDEContent:   pdeContent,
		FileName:     processing.FileName,
		OriginalName: processing.OriginalName,
		CanvasID:     processing.CanvasID,
		IsPreview:    false,
	}

	// リクエストをJSON化
	requestBody, err := json.Marshal(request)
	if err != nil {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("リクエストのJSONエンコードに失敗しました: %v", err)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("リクエストのJSONエンコードに失敗しました: %v", err)
	}

	// Lambda関数を呼び出し
	resp, err := s.httpClient.Post(lambdaURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("Lambda関数の呼び出しに失敗しました: %v", err)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("Lambda関数の呼び出しに失敗しました: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスステータスを確認
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("Lambda関数がエラーを返しました: HTTP %d", resp.StatusCode)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("Lambda関数がエラーを返しました: HTTP %d", resp.StatusCode)
	}

	// レスポンスをパース
	var lambdaResponse PDEConversionResponse
	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(&lambdaResponse); err != nil {
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

	// JSファイルを保存
	jsFileName := strings.TrimSuffix(processing.FileName, filepath.Ext(processing.FileName)) + ".js"
	jsDir := filepath.Join(s.config.Storage.UploadDir, "js")

	// JSディレクトリを作成
	if err := os.MkdirAll(jsDir, 0755); err != nil {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("JSディレクトリの作成に失敗しました: %v", err)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("JSディレクトリの作成に失敗しました: %v", err)
	}

	// JSファイルを書き込み
	jsPath := filepath.Join(jsDir, jsFileName)
	if err := os.WriteFile(jsPath, []byte(lambdaResponse.JSContent), 0644); err != nil {
		processing.Status = "error"
		processing.ErrorMessage = fmt.Sprintf("JSファイルの保存に失敗しました: %v", err)
		_ = s.processingRepo.Update(processing)
		return fmt.Errorf("JSファイルの保存に失敗しました: %v", err)
	}

	// 処理成功を記録
	processing.Status = "processed"
	processing.JSPath = fmt.Sprintf("/uploads/js/%s", jsFileName)
	processing.ErrorMessage = ""

	if err := s.processingRepo.Update(processing); err != nil {
		return fmt.Errorf("処理結果の更新に失敗しました: %v", err)
	}

	return nil
}

// InvokePreviewConversion プレビュー用のPDE変換を実行
func (s *lambdaService) InvokePreviewConversion(preview *PreviewProcessing) error {
	// Lambda関数のエンドポイント
	lambdaURL := s.config.AWS.LambdaEndpoint
	if lambdaURL == "" {
		return fmt.Errorf("Lambda関数のエンドポイントが設定されていません")
	}

	// リクエスト構築
	request := PDEConversionRequest{
		ProcessingID: 0, // プレビューのためのダミーID
		PDEContent:   preview.PDEContent,
		FileName:     preview.FileName,
		OriginalName: preview.OriginalName,
		CanvasID:     preview.CanvasID,
		IsPreview:    true,
	}

	// リクエストをJSON化
	requestBody, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("リクエストのJSONエンコードに失敗しました: %v", err)
	}

	// Lambda関数を呼び出し
	resp, err := s.httpClient.Post(lambdaURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		return fmt.Errorf("Lambda関数の呼び出しに失敗しました: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスステータスを確認
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("Lambda関数がエラーを返しました: HTTP %d", resp.StatusCode)
	}

	// プレビューモードではJSの返却は待たない
	return nil
}
