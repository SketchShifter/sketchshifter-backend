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

	// URLを構築 - パスの後半部分だけを使用してURLを構築
	url := f.baseURL

	// uploads/ディレクトリからの相対パスを計算
	// まずdestPathから最後のuploadsディレクトリ以降を抽出
	uploadsIndex := -1
	for i := len(destPath) - 1; i >= 0; i-- {
		if i+7 <= len(destPath) && destPath[i:i+7] == "uploads" {
			uploadsIndex = i + 7 // "uploads"の後のインデックス
			break
		}
	}

	if uploadsIndex != -1 && uploadsIndex < len(destPath) {
		// uploadsディレクトリの後の部分を取得
		relativePath := destPath[uploadsIndex:]
		if len(relativePath) > 0 && relativePath[0] == filepath.Separator {
			relativePath = relativePath[1:] // 先頭のスラッシュを削除
		}
		// パスをURL形式に正規化（WindowsのパスをURLに適したものに変換）
		url = f.baseURL + "/" + filepath.ToSlash(relativePath)
	} else {
		// 単純にファイル名だけを使用
		fileName := filepath.Base(destPath)
		url = f.baseURL + "/" + fileName
	}

	return url, nil
}

// DeleteFile ファイルを削除
func (f *fileUtils) DeleteFile(path string) error {
	// まずパスからファイル名を抽出
	fileName := filepath.Base(path)

	// パスがURLの場合、ファイルパスに変換
	filePath := ""
	if len(path) > 0 && path[0] == '/' {
		// パスが/で始まる場合は相対パスとみなす
		filePath = filepath.Join(f.baseURL, path[1:])
	} else {
		// 絶対パスの場合はそのまま使用
		filePath = filepath.Join(f.baseURL, fileName)
	}

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
