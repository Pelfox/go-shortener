package services

import (
	"testing"
)

func BenchmarkUserService_CreateUserCookie(b *testing.B) {
	secret := []byte("super-secret-benchmark-key-123456789")
	service := NewUserService(secret)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.CreateUserCookie()
	}
}

func BenchmarkUserService_VerifyUserCookieValue(b *testing.B) {
	secret := []byte("super-secret-benchmark-key-123456789")
	service := NewUserService(secret)
	_, cookieValue := service.CreateUserCookie()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = service.VerifyUserCookieValue(cookieValue)
	}
}
