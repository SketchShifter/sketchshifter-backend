package utils

import (
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// FileUtils ファイル操作に関するインターフェース
type FileUtils interface {
	SaveFile(src io.Reader, destPath string) (string, error)
	DeleteFile(path string) error
}

// fileUtils FileUtilsの実装
type fileUtils struct {
	baseURL string
}

// NewFileUtils FileUtilsを作成
func NewFileUtils(baseURL string) FileUtils {
	return &fileUtils{
		baseURL: baseURL,
	}
}

// SaveFile ファイルを保存
func (f *fileUtils) SaveFile(src io.Reader, destPath string) (string, error) {
	// ディレクトリを確認
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}

	// ファイルを作成
	dest, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer dest.Close()

	// ファイルをコピー
	if _, err := io.Copy(dest, src); err != nil {
		return "", err
	}

	// ファイル名だけを取得
	fileName := filepath.Base(destPath)

	// URLを構築 - 常に単純な形式を使用
	url := "/uploads/" + fileName

	return url, nil
}

// DeleteFile ファイルを削除
func (f *fileUtils) DeleteFile(path string) error {
	// まずパスからファイル名を抽出
	fileName := filepath.Base(path)

	// ファイルパスを再構築
	filePath := filepath.Join(f.baseURL, fileName)

	// ファイルの存在を確認
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // ファイルが存在しなければ成功とみなす
	}

	// ファイルを削除
	return os.Remove(filePath)
}

// GenerateRandomString ランダムな文字列を生成
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	// 乱数生成器を初期化
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[r.Intn(len(charset))]
	}
	return string(b)
}
