package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"
)

func main() {
	// 環境変数の読み込み
	workerURL := os.Getenv("CLOUDFLARE_WORKER_URL")
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")

	if workerURL == "" || apiKey == "" {
		fmt.Println("環境変数が設定されていません")
		fmt.Println("CLOUDFLARE_WORKER_URL と CLOUDFLARE_API_KEY を設定してください")
		os.Exit(1)
	}

	fmt.Printf("Cloudflare Worker URL: %s\n", workerURL)
	fmt.Printf("APIキー: %s...%s\n", apiKey[:4], apiKey[len(apiKey)-4:])

	// ヘルスチェック
	fmt.Println("\n=== ヘルスチェック ===")
	healthURL := fmt.Sprintf("%s/health", workerURL)
	req, err := http.NewRequest("GET", healthURL, nil)
	if err != nil {
		fmt.Printf("リクエスト作成エラー: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("X-API-Key", apiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("ヘルスチェックエラー: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("ステータスコード: %d\n", resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("レスポンス読み取りエラー: %v\n", err)
	} else {
		var prettyJSON bytes.Buffer
		json.Indent(&prettyJSON, body, "", "  ")
		fmt.Printf("レスポンス: \n%s\n", prettyJSON.String())
	}

	// テストファイルアップロード
	fmt.Println("\n=== テストファイルアップロード ===")

	// テスト画像ファイルを作成
	testFilePath := "/tmp/test_image.jpg"
	if err := createTestImage(testFilePath); err != nil {
		fmt.Printf("テスト画像作成エラー: %v\n", err)
		os.Exit(1)
	}
	defer os.Remove(testFilePath)

	// ファイルを開く
	file, err := os.Open(testFilePath)
	if err != nil {
		fmt.Printf("ファイルオープンエラー: %v\n", err)
		os.Exit(1)
	}
	defer file.Close()

	// アップロードリクエスト作成
	body = &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// ファイル部分
	part, err := writer.CreateFormFile("file", "test_image.jpg")
	if err != nil {
		fmt.Printf("マルチパート作成エラー: %v\n", err)
		os.Exit(1)
	}

	if _, err := io.Copy(part, file); err != nil {
		fmt.Printf("ファイルコピーエラー: %v\n", err)
		os.Exit(1)
	}

	// タイプフィールド
	writer.WriteField("type", "original")
	writer.WriteField("fileName", "test_image.jpg")
	writer.Close()

	// リクエスト送信
	uploadURL := fmt.Sprintf("%s/upload", workerURL)
	req, err = http.NewRequest("POST", uploadURL, body)
	if err != nil {
		fmt.Printf("アップロードリクエスト作成エラー: %v\n", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("X-API-Key", apiKey)

	resp, err = client.Do(req)
	if err != nil {
		fmt.Printf("アップロードエラー: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	fmt.Printf("アップロードステータス: %d\n", resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("アップロードレスポンス読み取りエラー: %v\n", err)
	} else {
		var prettyJSON bytes.Buffer
		json.Indent(&prettyJSON, body, "", "  ")
		fmt.Printf("アップロードレスポンス: \n%s\n", prettyJSON.String())
	}

	fmt.Println("\n=== テスト完了 ===")
}

// 簡単なテスト画像を作成
func createTestImage(path string) error {
	// 1x1ピクセルのJPEG画像データ
	jpegData := []byte{
		0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x01, 0x00, 0x48,
		0x00, 0x48, 0x00, 0x00, 0xFF, 0xDB, 0x00, 0x43, 0x00, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
		0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xC2, 0x00, 0x0B, 0x08, 0x00, 0x01, 0x00,
		0x01, 0x01, 0x11, 0x00, 0xFF, 0xC4, 0x00, 0x14, 0x10, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xDA, 0x00, 0x08, 0x01, 0x01,
		0x00, 0x01, 0x3F, 0x10, 0xFF, 0xD9,
	}

	return os.WriteFile(path, jpegData, 0644)
}
