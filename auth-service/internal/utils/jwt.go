package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT Claims 结构
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTUtil JWT 工具
type JWTUtil struct {
	secret          []byte
	accessTokenTTL  time.Duration
	refreshTokenTTL time.Duration
}

// NewJWTUtil 创建 JWT 工具
func NewJWTUtil(secret string, accessTokenTTL, refreshTokenTTL int64) *JWTUtil {
	return &JWTUtil{
		secret:          []byte(secret),
		accessTokenTTL:  time.Duration(accessTokenTTL) * time.Second,
		refreshTokenTTL: time.Duration(refreshTokenTTL) * time.Second,
	}
}

// GenerateToken 生成 Access Token
func (j *JWTUtil) GenerateToken(userID string, email, role string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(j.accessTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "vasset-auth",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

// GenerateRefreshToken 生成 Refresh Token
func (j *JWTUtil) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(now.Add(j.refreshTokenTTL)),
		IssuedAt:  jwt.NewNumericDate(now),
		Issuer:    "vasset-auth",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(j.secret)
}

// ParseToken 解析和验证 Token
func (j *JWTUtil) ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.secret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// GetAccessTokenTTL 获取 Access Token 有效期(秒)
func (j *JWTUtil) GetAccessTokenTTL() int64 {
	return int64(j.accessTokenTTL.Seconds())
}

// GetRefreshTokenTTL 获取 Refresh Token 有效期(秒)
func (j *JWTUtil) GetRefreshTokenTTL() int64 {
	return int64(j.refreshTokenTTL.Seconds())
}
