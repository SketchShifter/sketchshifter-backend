// internal/utils/cloudflare_test.go
package utils

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// CloudflareR2への接続をテストする（環境変数が設定されている場合のみ実行）
func TestCloudflareConnection(t *testing.T) {
	workerURL := os.Getenv("CLOUDFLARE_WORKER_URL")
	apiKey := os.Getenv("CLOUDFLARE_API_KEY")

	if workerURL == "" || apiKey == "" {
		t.Skip("CLOUDFLARE_WORKER_URL または CLOUDFLARE_API_KEY が設定されていないため、テストをスキップします")
	}

	// Cloudflareワーカーのヘルスエンドポイントにリクエスト
	resp, err := makeRequest("GET", workerURL+"/health", apiKey, nil)
	if err != nil {
		t.Fatalf("Cloudflareワーカーへの接続に失敗しました: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("予期しないステータスコード: %d", resp.StatusCode)
	}

	// レスポンスボディを出力
	body, err := readResponseBody(resp)
	if err != nil {
		t.Fatalf("レスポンスの読み取りに失敗しました: %v", err)
	}

	fmt.Printf("Cloudflareワーカーのヘルスチェック結果: %s\n", body)
}

// HTTPリクエストを作成して送信する
func makeRequest(method, url, apiKey string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	return client.Do(req)
}

// レスポンスボディを読み取る
func readResponseBody(resp *http.Response) (string, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
