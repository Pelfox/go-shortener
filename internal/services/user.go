package services

import (
	"errors"
	"fmt"

	"github.com/Pelfox/go-shortener/pkg"
)

var (
	ErrCookieInvalid = errors.New("user ID cookie is invalid")
)

type UserService struct {
	secret []byte
}

func NewUserService(secret []byte) *UserService {
	return &UserService{
		secret: secret,
	}
}

func (s *UserService) CreateUserCookie() (string, string) {
	userID := pkg.GenerateUserID()
	signedUserID := pkg.SignUserCookie(userID, s.secret)

	cookieValue := fmt.Sprintf("%s.%s", userID, signedUserID)
	return userID, cookieValue
}

func (s *UserService) VerifyUserCookieValue(value string) (string, error) {
	userID, valid := pkg.VerifyUserCookie(value, s.secret)
	if !valid {
		return "", ErrCookieInvalid
	}
	return userID, nil
}
