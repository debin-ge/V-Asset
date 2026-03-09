package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	pb "vasset/api-gateway/proto"
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
		cachedData, cacheErr := redisClient.HGetAll(ctx, cacheKey).Result()
		if cacheErr == nil && len(cachedData) > 0 {
			userID := cachedData["user_id"]
			if userID != "" {
				return &AuthClaims{
					UserID: userID,
					Email:  cachedData["email"],
					Role:   cachedData["role"],
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
		redisClient.HSet(ctx, cacheKey, map[string]interface{}{
			"user_id": resp.UserId,
			"email":   resp.Email,
			"role":    resp.Role,
		})
		redisClient.Expire(ctx, cacheKey, 5*time.Minute)
	}

	return &AuthClaims{
		UserID: resp.UserId,
		Email:  resp.Email,
		Role:   resp.Role,
		Token:  token,
	}, nil
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
