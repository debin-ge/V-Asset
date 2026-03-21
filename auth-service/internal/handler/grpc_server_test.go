package handler

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestAuthStatusErrorMapsValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code codes.Code
	}{
		{name: "duplicate email", err: assertError("邮箱已被注册"), code: codes.AlreadyExists},
		{name: "invalid email", err: assertError("邮箱格式不正确"), code: codes.InvalidArgument},
		{name: "nickname length", err: assertError("昵称长度必须在2-30个字符之间"), code: codes.InvalidArgument},
		{name: "user missing", err: assertError("用户不存在"), code: codes.NotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mapped := authStatusError(tt.err)
			if status.Code(mapped) != tt.code {
				t.Fatalf("expected %s, got %s", tt.code, status.Code(mapped))
			}
		})
	}
}

func TestAuthStatusErrorPreservesStatusErrors(t *testing.T) {
	original := status.Error(codes.PermissionDenied, "forbidden")
	if mapped := authStatusError(original); status.Code(mapped) != codes.PermissionDenied {
		t.Fatalf("expected status code to be preserved, got %s", status.Code(mapped))
	}
}

func assertError(message string) error {
	return errors.New(message)
}
