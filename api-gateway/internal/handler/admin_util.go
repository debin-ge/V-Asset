package handler

import (
	"log"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"vasset/api-gateway/internal/models"
)

func grpcErrorMessage(err error) string {
	if err == nil {
		return "request failed, please try again later"
	}

	st, ok := status.FromError(err)
	if !ok {
		return "request failed, please try again later"
	}

	message := strings.TrimSpace(st.Message())
	switch st.Code() {
	case codes.InvalidArgument, codes.FailedPrecondition, codes.OutOfRange,
		codes.NotFound, codes.AlreadyExists, codes.ResourceExhausted,
		codes.Unauthenticated, codes.PermissionDenied:
		if message != "" {
			return message
		}
	case codes.Unavailable:
		return "service temporarily unavailable"
	case codes.DeadlineExceeded:
		return "request timed out, please try again later"
	}

	return "request failed, please try again later"
}

func writeGRPCError(c *gin.Context, err error) {
	log.Printf("[Gateway] gRPC request failed: code=%s err=%v", status.Code(err), err)

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
