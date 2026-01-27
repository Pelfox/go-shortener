package pkg

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"
)

// SignUserCookie подписывает содержимое для Cookie, в котором содержится ID пользователя.
func SignUserCookie(userID string, secret []byte) string {
	signer := hmac.New(sha256.New, secret)
	signer.Write([]byte(userID))
	signature := signer.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(signature)
}

// VerifyUserCookie проверяет подлинность значения Cookie, содержащего ID пользователя.
// Ожидается, что значение аргумента `value` будет в следующем формате: `userID.signature`.
func VerifyUserCookie(value string, secret []byte) (string, bool) {
	cookieParts := strings.SplitN(value, ".", 2)
	if len(cookieParts) != 2 {
		return "", false
	}

	userID := cookieParts[0]
	expectedSignature := SignUserCookie(userID, secret)

	comparison := subtle.ConstantTimeCompare([]byte(expectedSignature), []byte(cookieParts[1]))
	if comparison != 1 {
		return "", false
	}

	return userID, true
}
