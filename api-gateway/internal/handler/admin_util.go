package handler

import (
	"strings"

	"google.golang.org/grpc/status"
)

func grpcErrorMessage(err error) string {
	if err == nil {
		return ""
	}

	st, ok := status.FromError(err)
	if !ok {
		return err.Error()
	}

	message := strings.TrimSpace(st.Message())
	if message == "" {
		return err.Error()
	}
	return message
}
