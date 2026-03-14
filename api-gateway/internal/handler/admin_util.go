package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/api-gateway/internal/models"
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

func writeGRPCError(c *gin.Context, err error) {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange:
		models.BadRequest(c, grpcErrorMessage(err))
	case codes.NotFound:
		models.NotFound(c, grpcErrorMessage(err))
	case codes.AlreadyExists:
		models.Conflict(c, grpcErrorMessage(err))
	case codes.ResourceExhausted:
		models.Forbidden(c, grpcErrorMessage(err))
	case codes.Unauthenticated:
		models.Unauthorized(c, grpcErrorMessage(err))
	case codes.PermissionDenied:
		models.Forbidden(c, grpcErrorMessage(err))
	default:
		models.InternalError(c, grpcErrorMessage(err))
	}
}
