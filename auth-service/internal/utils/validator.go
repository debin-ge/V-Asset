package utils

import (
	"errors"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return errors.New("邮箱不能为空")
	}
	if !emailRegex.MatchString(email) {
		return errors.New("邮箱格式不正确")
	}
	return nil
}

// ValidateNickname 验证昵称
func ValidateNickname(nickname string) error {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return nil // 昵称可选
	}

	runeCount := len([]rune(nickname))
	if runeCount < 2 || runeCount > 30 {
		return errors.New("昵称长度必须在2-30个字符之间")
	}

	return nil
}
