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
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// UploadController アップロードに関するコントローラー
type UploadController struct {
	uploadDir        string
	cloudflareBucket string
	cfWorkerURL      string
	cfAPIKey         string
}

// NewUploadController UploadControllerを作成
func NewUploadController(uploadDir, cfWorkerURL, cfAPIKey, cloudflareBucket string) *UploadController {
	return &UploadController{
		uploadDir:        uploadDir,
		cloudflareBucket: cloudflareBucket,
		cfWorkerURL:      cfWorkerURL,
		cfAPIKey:         cfAPIKey,
	}
}

// CFUploadResponse Cloudflareレスポンス
type CFUploadResponse struct {
	Success bool   `json:"success"`
	Path    string `json:"path"`
	URL     string `json:"url"`
	Error   string `json:"error"`
}

// UploadFile ファイルをアップロード
func (c *UploadController) UploadFile(ctx *gin.Context) {
	// マルチパートフォームを解析
	if err := ctx.Request.ParseMultipartForm(32 << 20); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "マルチパートフォームの解析に失敗しました"})
		return
	}

	// ファイルを取得
	file, header, err := ctx.Request.FormFile("file")
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "ファイルが必要です"})
		return
	}
	defer file.Close()

	// ユーザーからの入力を取得
	// fileName := generateFileName(header.Filename)
	fileName := header.Filename

	// デバッグ出力
	fmt.Printf("ファイルアップロード: %s, サイズ: %d\n", fileName, header.Size)

	// Cloudflare Workersを使用してR2にアップロード
	if err := c.uploadToCloudflare(ctx, file, header); err != nil {
		// Cloudflareアップロード失敗時はローカルストレージにフォールバック
		fmt.Printf("Cloudflareアップロード失敗: %v, ローカルにフォールバック\n", err)
		localPath, err := c.saveToLocalStorage(file, header, fileName)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("ファイル保存に失敗しました: %v", err)})
			return
		}

		// 成功レスポンスを返す（ローカルパス）
		ctx.JSON(http.StatusOK, gin.H{
			"success":  true,
			"file_url": localPath,
			"message":  "ファイルがローカルストレージにアップロードされました",
		})
		return
	}

	// 正常完了（Cloudflareアップロード成功）
	ctx.JSON(http.StatusOK, gin.H{
		"success":  true,
		"file_url": fmt.Sprintf("%s/file/images/%s", c.cfWorkerURL, fileName),
		"message":  "ファイルがCloudflare R2にアップロードされました",
	})
}

// Cloudflare WorkersのR2ストレージにアップロード
func (c *UploadController) uploadToCloudflare(ctx *gin.Context, file multipart.File, header *multipart.FileHeader) error {
	// リクエストのための新しいFormDataを作成
	// body := &bytes.Buffer{}
	writer := http.Client{}

	// ファイルを保存
	tempPath := filepath.Join(os.TempDir(), header.Filename)
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("一時ファイルの作成に失敗しました: %v", err)
	}
	defer tempFile.Close()
	defer os.Remove(tempPath) // 一時ファイルを削除

	// ファイル内容をコピー
	if _, err = io.Copy(tempFile, file); err != nil {
		return fmt.Errorf("ファイルの書き込みに失敗しました: %v", err)
	}
	tempFile.Close() // 書き込み後にクローズして読み込みモードで再オープン

	// ファイルを再度開く
	tempFile, err = os.Open(tempPath)
	if err != nil {
		return fmt.Errorf("一時ファイルを開けません: %v", err)
	}
	defer tempFile.Close()

	// マルチパートリクエストを準備
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/upload", c.cfWorkerURL), nil)
	if err != nil {
		return fmt.Errorf("リクエスト作成エラー: %v", err)
	}

	// フォームデータを作成
	formData := &bytes.Buffer{}
	formWriter := multipart.NewWriter(formData)

	// ファイルフィールドを追加
	filePart, err := formWriter.CreateFormFile("file", header.Filename)
	if err != nil {
		return fmt.Errorf("フォームファイル作成エラー: %v", err)
	}

	// ファイル内容をコピー
	if _, err = io.Copy(filePart, tempFile); err != nil {
		return fmt.Errorf("ファイルコピーエラー: %v", err)
	}

	// フォームデータを閉じる
	formWriter.Close()

	// リクエストを設定
	req, err = http.NewRequest("POST", fmt.Sprintf("%s/upload", c.cfWorkerURL), formData)
	if err != nil {
		return fmt.Errorf("リクエスト作成エラー: %v", err)
	}
	req.Header.Set("Content-Type", formWriter.FormDataContentType())

	// API Keyがあれば設定
	if c.cfAPIKey != "" {
		req.Header.Set("X-API-Key", c.cfAPIKey)
	}

	// リクエストを送信
	resp, err := writer.Do(req)
	if err != nil {
		return fmt.Errorf("Cloudflareリクエスト失敗: %v", err)
	}
	defer resp.Body.Close()

	// レスポンスボディを読み込む
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("レスポンス読み込みエラー: %v", err)
	}

	fmt.Printf("Cloudflareレスポンス: %s\n", string(respBody))

	// ステータスコードをチェック
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Cloudflareアップロード失敗: HTTP %d, レスポンス: %s", resp.StatusCode, string(respBody))
	}

	// レスポンスをパース
	var cfResp CFUploadResponse
	if err := json.Unmarshal(respBody, &cfResp); err != nil {
		return fmt.Errorf("JSONパースエラー: %v", err)
	}

	// 成功を確認
	if !cfResp.Success {
		return fmt.Errorf("Cloudflareアップロードエラー: %s", cfResp.Error)
	}

	return nil
}

// ローカルストレージにファイルを保存
func (c *UploadController) saveToLocalStorage(file multipart.File, header *multipart.FileHeader, fileName string) (string, error) {
	// アップロードディレクトリを確認
	uploadDir := filepath.Join(c.uploadDir, "original")
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", fmt.Errorf("ディレクトリの作成に失敗しました: %v", err)
	}

	// ファイルパスを作成
	filePath := filepath.Join(uploadDir, fileName)

	// ファイルを作成
	out, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルの作成に失敗しました: %v", err)
	}
	defer out.Close()

	// ファイルをシーク
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("ファイルのシークに失敗しました: %v", err)
	}

	// ファイルをコピー
	if _, err = io.Copy(out, file); err != nil {
		return "", fmt.Errorf("ファイルの書き込みに失敗しました: %v", err)
	}

	// 相対URLを返す
	return fmt.Sprintf("/uploads/original/%s", fileName), nil
}

// ファイル名を生成
func generateFileName(originalName string) string {
	timestamp := time.Now().Unix()
	randomString := fmt.Sprintf("%09d", time.Now().Nanosecond())[0:8]
	extension := filepath.Ext(originalName)
	baseName := strings.TrimSuffix(filepath.Base(originalName), extension)
	sanitizedName := sanitizeFileName(baseName)

	return fmt.Sprintf("%d_%s_%s%s", timestamp, randomString, sanitizedName, extension)
}

// ファイル名を安全な形式に変換
func sanitizeFileName(name string) string {
	// 不正な文字を置換
	name = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, name)

	// 長すぎる名前を切り詰める
	if len(name) > 50 {
		name = name[:50]
	}

	return name
}
