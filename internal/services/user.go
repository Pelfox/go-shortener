package services

import (
	"errors"

	"github.com/Pelfox/go-shortener/pkg"
)

var (
	// ErrCookieInvalid указывает, что переданное значение куки
	// пользовательского ID не является корректным.
	ErrCookieInvalid = errors.New("user ID cookie is invalid")
)

// generate:reset
// UserService реализует логику простейшей авторизации пользователей.
type UserService struct {
	secret []byte
}

// NewUserService создаёт и возвращает новый объект UserService.
func NewUserService(secret []byte) *UserService {
	return &UserService{
		secret: secret,
	}
}

// CreateUserCookie создаёт новый пользовательский ID, и подписывает его
// используя предоставленный ранее secret и HMAC функции. Данный метод
// возвращает сам ID, а также значение для Cookie, разделённое через точку.
func (s *UserService) CreateUserCookie() (string, string) {
	userID := pkg.GenerateUserID()
	signedUserID := pkg.SignUserCookie(userID, s.secret)

	cookieValue := userID + "." + signedUserID
	return userID, cookieValue
}

// VerifyUserCookieValue проверяет, что указанное значение Cookie является
// валидным пользовательским ID, и соответствует указанной в нём сигнатуре.
// Возвращает пользовательский ID в случае корректного значения.
func (s *UserService) VerifyUserCookieValue(value string) (string, error) {
	userID, valid := pkg.VerifyUserCookie(value, s.secret)
	if !valid {
		return "", ErrCookieInvalid
	}
	return userID, nil
}
