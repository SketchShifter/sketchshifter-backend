package services

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"

	"github.com/SketchShifter/sketchshifter_backend/internal/config"
	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

// CloudinaryService Cloudinaryとの連携を管理するサービス
type CloudinaryService interface {
	UploadImage(file multipart.File, fileName string, compressionQuality int) (string, string, error)
	DeleteImage(publicID string) error
}

type cloudinaryService struct {
	cld *cloudinary.Cloudinary
	cfg *config.Config
}

// NewCloudinaryService CloudinaryServiceを作成
func NewCloudinaryService(cfg *config.Config) (CloudinaryService, error) {
	cld, err := cloudinary.NewFromParams(
		cfg.Cloudinary.CloudName,
		cfg.Cloudinary.APIKey,
		cfg.Cloudinary.APISecret,
	)
	if err != nil {
		return nil, err
	}

	return &cloudinaryService{
		cld: cld,
		cfg: cfg,
	}, nil
}

// UploadImage 画像をアップロード
func (s *cloudinaryService) UploadImage(file multipart.File, fileName string, compressionQuality int) (string, string, error) {
	// ファイルデータを読み込み
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(file); err != nil {
		return "", "", fmt.Errorf("ファイルの読み込みに失敗しました: %v", err)
	}

	// アップロードパラメータを設定
	uploadParams := uploader.UploadParams{
		Folder:       s.cfg.Cloudinary.Folder,
		PublicID:     fileName,
		ResourceType: "image",
		// 圧縮設定
		Transformation: fmt.Sprintf("q_%d", compressionQuality),
	}

	// アップロード
	ctx := context.Background()
	result, err := s.cld.Upload.Upload(ctx, buf, uploadParams)

	if err != nil {
		return "", "", fmt.Errorf("Cloudinaryへのアップロードに失敗しました: %v", err)
	}

	return result.PublicID, result.SecureURL, nil
}

// DeleteImage 画像を削除
func (s *cloudinaryService) DeleteImage(publicID string) error {
	if publicID == "" {
		return nil
	}

	// Cloudinaryから画像を削除
	_, err := s.cld.Upload.Destroy(context.Background(), uploader.DestroyParams{
		PublicID: publicID,
	})

	if err != nil {
		return fmt.Errorf("Cloudinaryからの削除に失敗しました: %v", err)
	}

	return nil
}
