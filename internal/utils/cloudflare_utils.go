// internal/utils/cloudflare_utils.go
package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
)

// CloudflareUtils Cloudflare R2アクセスユーティリティ
type CloudflareUtils interface {
	UploadFile(file multipart.File, fileName string, fileType string) (string, error)
}

type cloudflareUtils struct {
	workerURL string
	apiKey    string
}

// NewCloudflareUtils CloudflareUtilsを作成
func NewCloudflareUtils() CloudflareUtils {
	return &cloudflareUtils{
		workerURL: os.Getenv("CLOUDFLARE_WORKER_URL"),
		apiKey:    os.Getenv("CLOUDFLARE_API_KEY"),
	}
}

// UploadFile ファイルをCloudflare R2にアップロード
func (c *cloudflareUtils) UploadFile(file multipart.File, fileName string, fileType string) (string, error) {
	// 環境変数チェック
	if c.workerURL == "" || c.apiKey == "" {
		return "", fmt.Errorf("CLOUDFLARE_WORKER_URL または CLOUDFLARE_API_KEY が設定されていません")
	}

	// ファイルをリセット
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", err
	}

	// ファイル内容を読み込む
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	// マルチパートフォームデータを作成
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// ファイルフィールドを追加
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", err
	}

	if _, err := part.Write(fileBytes); err != nil {
		return "", err
	}

	// タイプフィールドを追加
	if err := writer.WriteField("type", fileType); err != nil {
		return "", err
	}

	// ファイル名フィールドを追加
	if err := writer.WriteField("fileName", fileName); err != nil {
		return "", err
	}

	writer.Close()

	// Cloudflare Workerにリクエスト
	req, err := http.NewRequest("POST", c.workerURL+"/upload", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", c.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// エラーレスポンスの内容を読み取り
		errorBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Cloudflare Workerが失敗しました: %s - %s", resp.Status, string(errorBody))
	}

	// レスポンスをパース
	var result struct {
		Success bool   `json:"success"`
		Path    string `json:"path"`
		URL     string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if !result.Success {
		return "", fmt.Errorf("Cloudflare Workerのレスポンスがエラーを示しています")
	}

	// URLを返す
	return c.workerURL + result.URL, nil
}
