package utils

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword 使用 bcrypt 加密密码
func HashPassword(password string, cost int) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// ComparePassword 验证密码
func ComparePassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePasswordStrength 验证密码强度
func ValidatePasswordStrength(password string, minLength int, requireUpper, requireLower, requireNumber, requireSpecial bool) error {
	if len(password) < minLength {
		return errors.New("密码长度不足")
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if requireUpper && !hasUpper {
		return errors.New("密码必须包含至少一个大写字母")
	}
	if requireLower && !hasLower {
		return errors.New("密码必须包含至少一个小写字母")
	}
	if requireNumber && !hasNumber {
		return errors.New("密码必须包含至少一个数字")
	}
	if requireSpecial && !hasSpecial {
		return errors.New("密码必须包含至少一个特殊字符")
	}

	return nil
}
