package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	pb "youdlp/api-gateway/proto"
)

// AuthClaims 是认证通过后可复用的身份信息。
type AuthClaims struct {
	UserID string
	Email  string
	Role   string
	Token  string
}

type tokenVerifier interface {
	VerifyToken(ctx context.Context, in *pb.VerifyTokenRequest, opts ...grpc.CallOption) (*pb.VerifyTokenResponse, error)
}

// AuthenticateToken 复用缓存和 Auth Service 完成 token 校验。
func AuthenticateToken(ctx context.Context, authClient tokenVerifier, redisClient *redis.Client, rawToken string) (*AuthClaims, error) {
	if authClient == nil {
		return nil, errors.New("auth service unavailable")
	}

	token, err := normalizeToken(rawToken)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	cacheKey := fmt.Sprintf("auth:token:%s", hashToken(token))
	if redisClient != nil {
		cachedData, cacheErr := redisClient.Get(ctx, cacheKey).Result()
		if cacheErr == nil && cachedData != "" {
			var claims AuthClaims
			if err := json.Unmarshal([]byte(cachedData), &claims); err == nil && claims.UserID != "" {
				return &AuthClaims{
					UserID: claims.UserID,
					Email:  claims.Email,
					Role:   claims.Role,
					Token:  token,
				}, nil
			}
		}
	}

	verifyCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	resp, err := authClient.VerifyToken(verifyCtx, &pb.VerifyTokenRequest{
		Token: token,
	})
	if err != nil {
		return nil, errors.New("token verification failed")
	}
	if !resp.Valid {
		return nil, errors.New("invalid token")
	}

	if redisClient != nil {
		claimsJSON, err := json.Marshal(AuthClaims{
			UserID: resp.UserId,
			Email:  resp.Email,
			Role:   resp.Role,
		})
		if err == nil {
			redisClient.Set(ctx, cacheKey, claimsJSON, 5*time.Minute)
		}
	}

	return &AuthClaims{
		UserID: resp.UserId,
		Email:  resp.Email,
		Role:   resp.Role,
		Token:  token,
	}, nil
}

func InvalidateTokenCache(ctx context.Context, redisClient *redis.Client, rawToken string) error {
	if redisClient == nil {
		return nil
	}

	token, err := normalizeToken(rawToken)
	if err != nil {
		return err
	}

	cacheKey := fmt.Sprintf("auth:token:%s", hashToken(token))
	return redisClient.Del(ctx, cacheKey).Err()
}

func normalizeToken(raw string) (string, error) {
	token := strings.TrimSpace(raw)
	if token == "" {
		return "", errors.New("missing authorization token")
	}

	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		token = strings.TrimSpace(token[7:])
	}
	if token == "" {
		return "", errors.New("empty token")
	}

	return token, nil
}
