package services

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/SketchShifter/sketchshifter_backend/internal/utils"
)

// FileService ファイルストレージに関するサービスインターフェース
type FileService interface {
	// 新しいファイルをアップロード
	UploadFile(file multipart.File, fileName, subDir string) (string, error)
	// ファイルを削除
	DeleteFile(filePath string) error
	// ファイルを取得
	GetFile(filePath string) ([]byte, string, error)
	// 一時ファイルを作成
	CreateTempFile(content []byte, extension string) (string, error)
	// プレビューURLを作成 (PDEファイルまたはコードから)
	CreatePreviewFile(file multipart.File, fileName, code string) (string, error)
}

// fileService FileServiceの実装
type fileService struct {
	config     *config.Config
	uploadRoot string
	baseURL    string
}

// NewFileService FileServiceを作成
func NewFileService(cfg *config.Config) FileService {
	uploadRoot := cfg.Storage.UploadDir

	// 基本的なアップロードディレクトリ構造を作成
	dirs := []string{
		uploadRoot,
		filepath.Join(uploadRoot, "original"),
		filepath.Join(uploadRoot, "preview"),
		filepath.Join(uploadRoot, "thumbnail"),
		filepath.Join(uploadRoot, "js"),
	}

	for _, dir := range dirs {
		_ = os.MkdirAll(dir, 0755)
	}

	// API URLからベースURL構築
	baseURL := "/uploads" // デフォルト

	return &fileService{
		config:     cfg,
		uploadRoot: uploadRoot,
		baseURL:    baseURL,
	}
}

// UploadFile ファイルをアップロード
func (s *fileService) UploadFile(file multipart.File, fileName, subDir string) (string, error) {
	if file == nil {
		return "", errors.New("ファイルが必要です")
	}

	// ディレクトリパスを作成
	dirPath := filepath.Join(s.uploadRoot, subDir)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("ディレクトリの作成に失敗しました: %v", err)
	}

	// ファイルパスを作成
	filePath := filepath.Join(dirPath, fileName)

	// ファイルを作成
	dest, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("ファイルの作成に失敗しました: %v", err)
	}
	defer dest.Close()

	// シーク位置をリセット
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("ファイルのシークに失敗しました: %v", err)
	}

	// ファイルをコピー
	if _, err := io.Copy(dest, file); err != nil {
		return "", fmt.Errorf("ファイルのコピーに失敗しました: %v", err)
	}

	// URLを構築 (例: /uploads/original/filename.jpg)
	url := fmt.Sprintf("%s/%s/%s", s.baseURL, subDir, fileName)

	return url, nil
}

// DeleteFile ファイルを削除
func (s *fileService) DeleteFile(filePath string) error {
	// パスをローカルファイルシステムのパスに変換
	localPath := s.convertURLToLocalPath(filePath)
	if localPath == "" {
		return fmt.Errorf("無効なファイルパス: %s", filePath)
	}

	// ファイルの存在確認
	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		// ファイルが存在しない場合はエラーを返さない
		return nil
	}

	// ファイルを削除
	return os.Remove(localPath)
}

// GetFile ファイルを取得
func (s *fileService) GetFile(filePath string) ([]byte, string, error) {
	// パスをローカルファイルシステムのパスに変換
	localPath := s.convertURLToLocalPath(filePath)
	if localPath == "" {
		return nil, "", fmt.Errorf("無効なファイルパス: %s", filePath)
	}

	// ファイルを読み込み
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, "", fmt.Errorf("ファイルの読み込みに失敗しました: %v", err)
	}

	// Content-Typeを推定
	contentType := s.getContentTypeFromFilename(filePath)

	return data, contentType, nil
}

// CreateTempFile 一時ファイルを作成
func (s *fileService) CreateTempFile(content []byte, extension string) (string, error) {
	// 一時ディレクトリパスを作成
	tempDir := filepath.Join(s.uploadRoot, "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return "", fmt.Errorf("一時ディレクトリの作成に失敗しました: %v", err)
	}

	// ランダムなファイル名を生成
	fileName := fmt.Sprintf("%d_%s%s", time.Now().Unix(), utils.GenerateRandomString(8), extension)
	filePath := filepath.Join(tempDir, fileName)

	// ファイルを作成
	err := os.WriteFile(filePath, content, 0644)
	if err != nil {
		return "", fmt.Errorf("一時ファイルの作成に失敗しました: %v", err)
	}

	// URLを構築
	url := fmt.Sprintf("%s/temp/%s", s.baseURL, fileName)

	// 24時間後に自動削除するゴルーチンを起動
	go func() {
		time.Sleep(24 * time.Hour)
		_ = os.Remove(filePath)
	}()

	return url, nil
}

// CreatePreviewFile プレビューファイルを作成
func (s *fileService) CreatePreviewFile(file multipart.File, fileName, code string) (string, error) {
	// ファイルかコードのいずれかが必要
	if file == nil && code == "" {
		return "", errors.New("ファイルまたはコードが必要です")
	}

	// プレビューディレクトリパスを作成
	previewDir := filepath.Join(s.uploadRoot, "preview")
	if err := os.MkdirAll(previewDir, 0755); err != nil {
		return "", fmt.Errorf("プレビューディレクトリの作成に失敗しました: %v", err)
	}

	// タイムスタンプとランダム文字列を含むファイル名を生成
	timeStamp := time.Now().Unix()
	randomStr := utils.GenerateRandomString(8)

	var previewPath string

	if file != nil {
		// ファイルの場合
		previewFileName := fmt.Sprintf("preview_%d_%s_%s", timeStamp, randomStr, fileName)
		previewPath = filepath.Join(previewDir, previewFileName)

		// ファイルを作成
		dest, err := os.Create(previewPath)
		if err != nil {
			return "", fmt.Errorf("プレビューファイルの作成に失敗しました: %v", err)
		}
		defer dest.Close()

		// シーク位置をリセット
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			return "", fmt.Errorf("ファイルのシークに失敗しました: %v", err)
		}

		// ファイルをコピー
		if _, err := io.Copy(dest, file); err != nil {
			return "", fmt.Errorf("ファイルのコピーに失敗しました: %v", err)
		}
	} else if code != "" {
		// コードの場合
		previewFileName := fmt.Sprintf("preview_%d_%s.pde", timeStamp, randomStr)
		previewPath = filepath.Join(previewDir, previewFileName)

		// コードをファイルに書き込み
		if err := os.WriteFile(previewPath, []byte(code), 0644); err != nil {
			return "", fmt.Errorf("プレビューファイルの作成に失敗しました: %v", err)
		}
	}

	// URLを構築
	url := fmt.Sprintf("%s/preview/%s", s.baseURL, filepath.Base(previewPath))

	// 1時間後に自動削除するゴルーチンを起動
	go func() {
		time.Sleep(1 * time.Hour)
		_ = os.Remove(previewPath)
	}()

	return url, nil
}

// convertURLToLocalPath URLをローカルファイルパスに変換
func (s *fileService) convertURLToLocalPath(urlPath string) string {
	// URLパスからパスを抽出
	trimmedPath := strings.TrimPrefix(urlPath, s.baseURL)
	trimmedPath = strings.TrimPrefix(trimmedPath, "/")

	// ルートパスからのフルパスを構築
	return filepath.Join(s.uploadRoot, trimmedPath)
}

// getContentTypeFromFilename ファイル名からContent-Typeを推定
func (s *fileService) getContentTypeFromFilename(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))

	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".pde":
		return "text/plain"
	case ".js":
		return "application/javascript"
	case ".html":
		return "text/html"
	case ".css":
		return "text/css"
	default:
		return "application/octet-stream"
	}
}
