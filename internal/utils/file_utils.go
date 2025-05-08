package utils

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// GenerateRandomString ランダムな文字列を生成
func GenerateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// 失敗した場合は時間ベースのフォールバック
		now := time.Now().UnixNano()
		for i := range bytes {
			bytes[i] = byte((now >> (i * 8)) & 0xff)
		}
	}
	return hex.EncodeToString(bytes)
}
