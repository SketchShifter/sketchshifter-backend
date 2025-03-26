package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
)

// CloudflareR2Client Cloudflare R2クライアント
type CloudflareR2Client struct {
	Config *config.CloudflareConfig
	Client *http.Client
}

// SignedURLResponse 署名付きURLのレスポンス
type SignedURLResponse struct {
	URL       string `json:"url"`
	Key       string `json:"key"`
	ExpiresAt int64  `json:"expiresAt"`
}

// NewCloudflareR2Client 新しいCloudflare R2クライアントを作成
func NewCloudflareR2Client(cfg *config.CloudflareConfig) *CloudflareR2Client {
	return &CloudflareR2Client{
		Config: cfg,
		Client: &http.Client{Timeout: 30 * time.Second},
	}
}

// GetSignedUploadURL アップロード用の署名付きURLを取得
func (c *CloudflareR2Client) GetSignedUploadURL(fileType, fileName string) (*SignedURLResponse, error) {
	requestURL := fmt.Sprintf("%s/getSignedUrl", c.Config.WorkerURL)
	
	data := map[string]interface{}{
		"bucket":   c.Config.R2BucketName,
		"method":   "PUT",
		"key":      fmt.Sprintf("%d_%s", time.Now().Unix(), fileName),
		"fileType": fileType,
	}
	
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	
	req, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Config.APIToken)
	
	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get signed URL: %s, status: %d", string(body), resp.StatusCode)
	}
	
	var result SignedURLResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	
	return &result, nil
}

// UploadFileToR2 ファイルをR2にアップロード (小さいファイル用)
func (c *CloudflareR2Client) UploadFileToR2(file multipart.File, fileHeader *multipart.FileHeader) (string, string, error) {
	// ファイルタイプを取得
	fileType := fileHeader.Header.Get("Content-Type")
	if fileType == "" {
		ext := filepath.Ext(fileHeader.Filename)
		switch ext {
		case ".jpg", ".jpeg":
			fileType = "image/jpeg"
		case ".png":
			fileType = "image/png"
		case ".gif":
			fileType = "image/gif"
		case ".pde":
			fileType = "text/plain"
		default:
			fileType = "application/octet-stream"
		}
	}
	
	// 署名付きURLを取得
	signedURL, err := c.GetSignedUploadURL(fileType, fileHeader.Filename)
	if err != nil {
		return "", "", err
	}
	
	// ファイルをバッファに読み込み
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return "", "", err
	}
	
	// ファイルをアップロード
	req, err := http.NewRequest("PUT", signedURL.URL, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return "", "", err
	}
	
	req.Header.Set("Content-Type", fileType)
	
	resp, err := c.Client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("failed to upload file: %s, status: %d", string(body), resp.StatusCode)
	}
	
	// 成功したらキーとURLを返す
	publicURL := fmt.Sprintf("%s/public/%s", c.Config.WorkerURL, signedURL.Key)
	return publicURL, signedURL.Key, nil
}
