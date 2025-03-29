package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
)

// PreviewController プレビュー生成に関するコントローラー
type PreviewController struct {
	uploadDir  string
	lambdaURL  string
	previewURL string
}

// NewPreviewController PreviewControllerを作成
func NewPreviewController(uploadDir, lambdaURL, previewURL string) *PreviewController {
	return &PreviewController{
		uploadDir:  uploadDir,
		lambdaURL:  lambdaURL,
		previewURL: previewURL,
	}
}

// PreviewRequest Lambda関数へのリクエストデータ
type PreviewRequest struct {
	ProcessingID uint   `json:"processingId"`
	PDEContent   string `json:"pdeContent"`
	FileName     string `json:"fileName"`
	CanvasID     string `json:"canvasId"`
}

// PreviewResponse Lambda関数からのレスポンスデータ
type PreviewResponse struct {
	Success      bool   `json:"success"`
	ProcessingID uint   `json:"processingId"`
	JSContent    string `json:"jsContent"`
	Message      string `json:"message"`
	PreviewURL   string `json:"previewUrl"`
}

// CreatePreview プレビューを生成
func (c *PreviewController) CreatePreview(ctx *gin.Context) {
	// マルチパートフォームを解析
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "マルチパートフォームの解析に失敗しました"})
		return
	}

	// ファイルとコードを取得
	file, header, _ := ctx.Request.FormFile("file")
	code := ctx.PostForm("code")

	// どちらかが必要
	if (file == nil || header == nil) && code == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ファイルまたはコードが必要です"})
		return
	}

	// ファイルが提供された場合はコードを読み込む
	if file != nil {
		defer file.Close()

		// ファイルを一時保存
		tempFileName := fmt.Sprintf("preview_%d_%s%s",
			time.Now().Unix(),
			randomString(8),
			filepath.Ext(header.Filename))

		previewPath, err := c.savePreviewFile(file, tempFileName)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("プレビューファイルの保存に失敗しました: %v", err)})
			return
		}

		// PDEファイルの場合はコードを読み込む
		if filepath.Ext(header.Filename) == ".pde" {
			// ファイルを再度開く
			savedFile, err := os.Open(previewPath)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError,
					gin.H{"error": fmt.Sprintf("ファイルの読み込みに失敗しました: %v", err)})
				return
			}
			defer savedFile.Close()

			// コードを読み込む
			codeBytes, err := io.ReadAll(savedFile)
			if err != nil {
				ctx.JSON(http.StatusInternalServerError,
					gin.H{"error": fmt.Sprintf("コードの読み込みに失敗しました: %v", err)})
				return
			}
			code = string(codeBytes)
		}

		// プレビューURLを返す
		previewURL := fmt.Sprintf("%s/%s", c.previewURL, tempFileName)
		ctx.JSON(http.StatusOK, gin.H{
			"success":     true,
			"preview_url": previewURL,
		})
		return
	}

	// コードだけが提供された場合はそれを保存
	if code != "" {
		// コードをファイルとして保存
		tempFileName := fmt.Sprintf("preview_%d_%s.pde", time.Now().Unix(), randomString(8))
		if err := c.saveCodeToFile(code, tempFileName); err != nil {
			ctx.JSON(http.StatusInternalServerError,
				gin.H{"error": fmt.Sprintf("コードの保存に失敗しました: %v", err)})
			return
		}

		// Lambda関数を呼び出す場合（オプション）
		jsContent, err := c.callLambdaFunction(code, tempFileName)
		if err != nil {
			// Lambda呼び出しに失敗してもプレビューは提供
			fmt.Printf("Lambda呼び出しエラー: %v\n", err)
		} else if jsContent != "" {
			// JSをファイルとして保存
			jsFileName := fmt.Sprintf("preview_%d_%s.js", time.Now().Unix(), randomString(8))
			if err := c.saveCodeToFile(jsContent, jsFileName); err != nil {
				fmt.Printf("JSファイル保存エラー: %v\n", err)
			}
		}

		// プレビューURLを返す
		previewURL := fmt.Sprintf("%s/%s", c.previewURL, tempFileName)
		ctx.JSON(http.StatusOK, gin.H{
			"success":     true,
			"preview_url": previewURL,
		})
		return
	}

	// ここには来ないはず
	ctx.JSON(http.StatusBadRequest, gin.H{"error": "無効なリクエスト"})
}

// ファイルをプレビューとして保存
func (c *PreviewController) savePreviewFile(file multipart.File, fileName string) (string, error) {
	// プレビューディレクトリを確認
	previewDir := filepath.Join(c.uploadDir, "preview")
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		return "", fmt.Errorf("ディレクトリの作成に失敗しました: %v", err)
	}

	// ファイルパスを作成
	filePath := filepath.Join(previewDir, fileName)

	// ファイルを作成
	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルの作成に失敗しました: %v", err)
	}
	defer out.Close()

	// ファイルをコピー
	if _, err = io.Copy(out, file); err != nil {
		return "", fmt.Errorf("ファイルの書き込みに失敗しました: %v", err)
	}

	// 自動削除タイマー
	go func() {
		time.Sleep(1 * time.Hour)
		os.Remove(filePath)
	}()

	return filePath, nil
}

// コードをファイルとして保存
func (c *PreviewController) saveCodeToFile(code, fileName string) error {
	// プレビューディレクトリを確認
	previewDir := filepath.Join(c.uploadDir, "preview")
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		return fmt.Errorf("ディレクトリの作成に失敗しました: %v", err)
	}

	// ファイルパスを作成
	filePath := filepath.Join(previewDir, fileName)

	// ファイルを作成して書き込み
	if err := os.WriteFile(filePath, []byte(code), 0644); err != nil {
		return fmt.Errorf("ファイルの書き込みに失敗しました: %v", err)
	}

	// 自動削除タイマー
	go func() {
		time.Sleep(1 * time.Hour)
		os.Remove(filePath)
	}()

	return nil
}

// Lambda関数を呼び出す
func (c *PreviewController) callLambdaFunction(code, fileName string) (string, error) {
	// Lambdaのエンドポイントが設定されていない場合はスキップ
	if c.lambdaURL == "" {
		return "", fmt.Errorf("Lambda URLが設定されていません")
	}

	// リクエストを作成
	req := PreviewRequest{
		ProcessingID: 0, // プレビューではIDは不要
		PDEContent:   code,
		FileName:     fileName,
		CanvasID:     fmt.Sprintf("preview_canvas_%s", randomString(8)),
	}

	// JSONにエンコード
	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("JSONエンコードエラー: %v", err)
	}

	// リクエストを送信
	resp, err := http.Post(c.lambdaURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("Lambda呼び出しエラー: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスを読み込む
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("レスポンス読み込みエラー: %v", err)
	}

	// レスポンスをパース
	var previewResp PreviewResponse
	if err := json.Unmarshal(respBody, &previewResp); err != nil {
		return "", fmt.Errorf("JSONパースエラー: %v, レスポンス: %s", err, string(respBody))
	}

	// 成功を確認
	if !previewResp.Success {
		return "", fmt.Errorf("Lambda処理エラー: %s", previewResp.Message)
	}

	return previewResp.JSContent, nil
}

// ランダム文字列を生成
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
