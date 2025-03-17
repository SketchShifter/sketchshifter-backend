package utils

import (
	"errors"
	"github.com/dgrijalva/jwt-go"
	"time"
)

var jwtSecret = []byte("your_jwt_secret_key")

// JWTClaims はJWTトークンのペイロード
type JWTClaims struct {
	UserID uint `json:"user_id"`
	jwt.StandardClaims
}

// GenerateJWT はユーザーIDからJWTトークンを生成する
func GenerateJWT(userID uint) (string, error) {
	// トークンの有効期限を設定（例: 24時間）
	expirationTime := time.Now().Add(24 * time.Hour)
	
	// クレームを作成
	claims := &JWTClaims{
		UserID: userID,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}
	
	// トークンを作成
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	// 署名して文字列化
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

// ValidateJWT はJWTトークンを検証しユーザーIDを返す
func ValidateJWT(tokenString string) (uint, error) {
	// トークンをパース
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 署名方法を確認
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return jwtSecret, nil
	})
	
	if err != nil {
		return 0, err
	}
	
	// クレームを取得
	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims.UserID, nil
	}
	
	return 0, errors.New("invalid token")
}
